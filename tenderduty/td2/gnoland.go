package tenderduty

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	
	dash "github.com/blockpane/tenderduty/v2/td2/dashboard"
)

// GnoStatus represents the status response from a Gnoland node
type GnoStatus struct {
	NodeInfo struct {
		Network string `json:"network"`
		Moniker string `json:"moniker"`
	} `json:"node_info"`
	SyncInfo struct {
		CatchingUp        bool   `json:"catching_up"`
		LatestBlockHeight string `json:"latest_block_height"`
		LatestBlockTime   string `json:"latest_block_time"`
	} `json:"sync_info"`
	ValidatorInfo struct {
		Address string `json:"address"`
		PubKey  struct {
			Type  string `json:"@type"`
			Value string `json:"value"`
		} `json:"pub_key"`
		VotingPower string `json:"voting_power"`
	} `json:"validator_info"`
}

// GnoValidatorSet represents the validator set from Gnoland
type GnoValidatorSet struct {
	Validators []struct {
		Address     string `json:"address"`
		PubKey      struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"pub_key"`
		VotingPower string `json:"voting_power"`
		ProposerPriority string `json:"proposer_priority"`
	} `json:"validators"`
}

// GnoBlockResponse represents the full block response from Gnoland
type GnoBlockResponse struct {
	BlockMeta struct {
		Header struct {
			Height string `json:"height"`
			Time   string `json:"time"`
		} `json:"header"`
	} `json:"block_meta"`
	Block struct {
		Header struct {
			Height string `json:"height"`
			Time   string `json:"time"`
		} `json:"header"`
		LastCommit struct {
			Precommits []struct {
				ValidatorAddress string `json:"validator_address"`
				Signature        string `json:"signature"`
			} `json:"precommits"`
		} `json:"last_commit"`
	} `json:"block"`
}

// IsGnolandChain checks if the chain is a Gnoland chain
func (cc *ChainConfig) IsGnolandChain() bool {
	return cc.ChainId == "test6" || strings.Contains(cc.ChainId, "gno")
}

// gnolandHTTPRequest makes HTTP requests to Gnoland RPC endpoints
func (cc *ChainConfig) gnolandHTTPRequest(ctx context.Context, endpoint string, params map[string]interface{}) ([]byte, error) {
	// Try all available nodes
	for _, node := range cc.Nodes {
		if node.down {
			continue
		}
		
		// Build the request URL
		url := strings.TrimSuffix(node.Url, "/") + endpoint
		
		// Create HTTP client with timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		
		// Make the request
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		
		// Add query parameters
		if params != nil {
			q := req.URL.Query()
			for k, v := range params {
				q.Add(k, fmt.Sprintf("%v", v))
			}
			req.URL.RawQuery = q.Encode()
		}
		
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		
		return body, nil
	}
	
	return nil, fmt.Errorf("no working nodes available")
}

// gnolandGetStatus gets the node status from Gnoland
func (cc *ChainConfig) gnolandGetStatus(ctx context.Context) (*GnoStatus, error) {
	body, err := cc.gnolandHTTPRequest(ctx, "/status", nil)
	if err != nil {
		return nil, err
	}
	
	// Parse the JSON response
	var result struct {
		Result GnoStatus `json:"result"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status response: %w", err)
	}
	
	return &result.Result, nil
}

// gnolandGetBlock gets a specific block from Gnoland
func (cc *ChainConfig) gnolandGetBlock(ctx context.Context, height string) (*GnoBlockResponse, error) {
	params := map[string]interface{}{
		"height": height,
	}
	
	body, err := cc.gnolandHTTPRequest(ctx, "/block", params)
	if err != nil {
		return nil, err
	}
	
	// DEBUG: Log first part of JSON response
	bodyStr := string(body)
	if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "..."
	}
	l(fmt.Sprintf("🔍 %s DEBUG: Block JSON response: %s", cc.ChainId, bodyStr))
	
	// Parse the JSON response
	var result struct {
		Result GnoBlockResponse `json:"result"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block response: %w", err)
	}
	
	return &result.Result, nil
}

// gnolandCheckValidatorSigned checks if our validator signed a specific block
func (cc *ChainConfig) gnolandCheckValidatorSigned(ctx context.Context, height string) int {
	// Get the block data
	block, err := cc.gnolandGetBlock(ctx, height)
	if err != nil {
		l(fmt.Sprintf("❌ %s error getting block %s: %s", cc.ChainId, height, err))
		return -1 // Unknown status
	}
	
	// DEBUG: Log raw block data to see what we're getting
	l(fmt.Sprintf("🔍 %s DEBUG: Block height=%s, precommits_count=%d", cc.ChainId, height, len(block.Block.LastCommit.Precommits)))
	
	// For Gnoland, validator address in signatures is the full address (with 'g' prefix)
	validatorAddr := cc.ValAddress
	
	// DEBUG: Log block signature info
	l(fmt.Sprintf("🔍 %s block %s has %d precommits, looking for validator %s", cc.ChainId, height, len(block.Block.LastCommit.Precommits), validatorAddr))
	
	// Check if our validator signed this block
	for i, sig := range block.Block.LastCommit.Precommits {
		sigPreview := sig.Signature
		if len(sigPreview) > 20 {
			sigPreview = sigPreview[:20] + "..."
		}
		l(fmt.Sprintf("🔍 %s precommit %d: validator=%s, signature=%s", cc.ChainId, i, sig.ValidatorAddress, sigPreview))
		if sig.ValidatorAddress == validatorAddr {
			if sig.Signature != "" {
				l(fmt.Sprintf("✅ %s validator %s SIGNED block %s", cc.ChainId, validatorAddr, height))
				return 3 // StatusSigned
			}
		}
	}
	
	l(fmt.Sprintf("❌ %s validator %s MISSED block %s", cc.ChainId, validatorAddr, height))
	return 0 // Statusmissed
}

// gnolandNewRpc sets up the RPC client for Gnoland
func (cc *ChainConfig) gnolandNewRpc() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Try each node until we find a working one
	for _, endpoint := range cc.Nodes {
		if endpoint.down {
			continue
		}
		
		// Test the connection
		status, err := cc.gnolandGetStatus(ctx)
		if err != nil {
			endpoint.down = true
			endpoint.lastMsg = fmt.Sprintf("❌ could not get status for %s: %s", cc.name, err)
			l(endpoint.lastMsg)
			continue
		}
		
		// Check chain ID
		if status.NodeInfo.Network != cc.ChainId {
			endpoint.down = true
			endpoint.lastMsg = fmt.Sprintf("chain id %s on %s does not match, expected %s", status.NodeInfo.Network, endpoint.Url, cc.ChainId)
			l(endpoint.lastMsg)
			continue
		}
		
		// Check if syncing
		if status.SyncInfo.CatchingUp {
			endpoint.down = true
			endpoint.syncing = true
			endpoint.lastMsg = fmt.Sprintf("🐢 node is not synced, skipping %s", endpoint.Url)
			l(endpoint.lastMsg)
			continue
		}
		
		// Success! Node is working
		endpoint.down = false
		endpoint.syncing = false
		cc.noNodes = false
		l(fmt.Sprintf("✅ %s connected to Gnoland node: %s", cc.name, endpoint.Url))
		return nil
	}
	
	cc.noNodes = true
	cc.lastError = "no usable Gnoland endpoints available for " + cc.ChainId
	
	// Update dashboard with error status
	if td.EnableDash {
		td.updateChan <- &dash.ChainStatus{
			MsgType:      "status",
			Name:         cc.name,
			ChainId:      cc.ChainId,
			Moniker:      "Unknown",
			Bonded:       false,
			Jailed:       false,
			Tombstoned:   false,
			Missed:       0,
			Window:       0,
			Nodes:        len(cc.Nodes),
			HealthyNodes: 0,
			ActiveAlerts: 1,
			Height:       0,
			LastError:    cc.lastError,
			Blocks:       cc.blocksResults,
		}
	}
	
	return fmt.Errorf(cc.lastError)
}

// gnolandGetValidatorInfo gets validator information from Gnoland
func (cc *ChainConfig) gnolandGetValidatorInfo(ctx context.Context) error {
	if cc.valInfo == nil {
		cc.valInfo = &ValInfo{}
	}
	
	// For Gnoland, we use the full validator address
	validatorAddr := cc.ValAddress
	
	// For now, set basic info
	cc.valInfo.Moniker = cc.ValAddress
	cc.valInfo.Bonded = true
	cc.valInfo.Jailed = false
	cc.valInfo.Tombstoned = false
	cc.valInfo.Missed = 0
	cc.valInfo.Window = 100 // Default window
	
	// Convert the validator address to consensus key format (remove 'g' prefix for consensus key)
	if validatorBytes, err := hex.DecodeString(strings.TrimPrefix(validatorAddr, "g")); err == nil {
		cc.valInfo.Conspub = validatorBytes
	}
	
	// Initialize blocksResults if needed
	if cc.blocksResults == nil {
		cc.blocksResults = make([]int, 512) // showBLocks = 512
		for i := range cc.blocksResults {
			cc.blocksResults[i] = -1 // Unknown status
		}
	}
	
	l(fmt.Sprintf("⚙️ Gnoland validator %s (%s) is being monitored", cc.ValAddress, cc.valInfo.Moniker))
	
	return nil
}

// gnolandMonitorBlocks starts monitoring blocks for Gnoland
func (cc *ChainConfig) gnolandMonitorBlocks(ctx context.Context) {
	l(fmt.Sprintf("⚙️ %s starting Gnoland block monitoring", cc.ChainId))
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	var lastProcessedHeight int64 = 0
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if cc.noNodes {
				continue
			}
			
			status, err := cc.gnolandGetStatus(ctx)
			if err != nil {
				l(fmt.Sprintf("❌ %s error getting status: %s", cc.name, err))
				continue
			}
			
			if !status.SyncInfo.CatchingUp {
				// Convert block height string to int64
				height := int64(0)
				if blockHeight, err := strconv.ParseInt(status.SyncInfo.LatestBlockHeight, 10, 64); err == nil {
					height = blockHeight
				}
				
				// Process new blocks
				if height > lastProcessedHeight {
					l(fmt.Sprintf("🧊 %s block %s", cc.ChainId, status.SyncInfo.LatestBlockHeight))
					
					// Check if our validator signed this block
					signStatus := cc.gnolandCheckValidatorSigned(ctx, status.SyncInfo.LatestBlockHeight)
					
					// Update blocksResults with the new block status
					cc.blocksResults = append([]int{signStatus}, cc.blocksResults[:len(cc.blocksResults)-1]...)
					
					// Update stats
					switch signStatus {
					case 0: // Statusmissed
						cc.valInfo.Missed++
						if cc.valInfo.Missed > cc.valInfo.Window {
							cc.valInfo.Missed = cc.valInfo.Window
						}
					case 3: // StatusSigned
						// Block signed successfully, no action needed
					}
					
					lastProcessedHeight = height
				}
				
				// Update dashboard with current status
				if td.EnableDash {
					// Count healthy nodes
					healthyNodes := 0
					for _, node := range cc.Nodes {
						if !node.down && !node.syncing {
							healthyNodes++
						}
					}
					
					td.updateChan <- &dash.ChainStatus{
						MsgType:      "status",
						Name:         cc.name,
						ChainId:      cc.ChainId,
						Moniker:      cc.valInfo.Moniker,
						Bonded:       cc.valInfo.Bonded,
						Jailed:       cc.valInfo.Jailed,
						Tombstoned:   cc.valInfo.Tombstoned,
						Missed:       cc.valInfo.Missed,
						Window:       cc.valInfo.Window,
						Nodes:        len(cc.Nodes),
						HealthyNodes: healthyNodes,
						ActiveAlerts: 0,
						Height:       height,
						LastError:    "",
						Blocks:       cc.blocksResults,
					}
				}
			}
		}
	}
}

// gnolandHealthCheck performs health checks for Gnoland nodes
func (cc *ChainConfig) gnolandHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, node := range cc.Nodes {
				go func(node *NodeConfig) {
					// Test this specific node
					client := &http.Client{Timeout: 10 * time.Second}
					url := strings.TrimSuffix(node.Url, "/") + "/status"
					
					req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
					if err != nil {
						if !node.down {
							node.down = true
							node.downSince = time.Now()
							node.lastMsg = fmt.Sprintf("❌ %s node %s is down: %s", cc.name, node.Url, err)
							l(node.lastMsg)
						}
						return
					}
					
					resp, err := client.Do(req)
					if err != nil {
						if !node.down {
							node.down = true
							node.downSince = time.Now()
							node.lastMsg = fmt.Sprintf("❌ %s node %s is down: %s", cc.name, node.Url, err)
							l(node.lastMsg)
						}
						return
					}
					defer resp.Body.Close()
					
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						if !node.down {
							node.down = true
							node.downSince = time.Now()
							node.lastMsg = fmt.Sprintf("❌ %s node %s is down: %s", cc.name, node.Url, err)
							l(node.lastMsg)
						}
						return
					}
					
					// Parse response
					var result struct {
						Result GnoStatus `json:"result"`
					}
					if err := json.Unmarshal(body, &result); err != nil {
						if !node.down {
							node.down = true
							node.downSince = time.Now()
							node.lastMsg = fmt.Sprintf("❌ %s node %s response error: %s", cc.name, node.Url, err)
							l(node.lastMsg)
						}
						return
					}
					
					// Check if syncing
					if result.Result.SyncInfo.CatchingUp {
						node.syncing = true
						node.lastMsg = fmt.Sprintf("🐢 %s node %s is syncing", cc.name, node.Url)
						return
					}
					
					// Node is healthy
					if node.down {
						node.lastMsg = ""
						node.wasDown = true
						l(fmt.Sprintf("🟢 %s node %s is healthy", cc.name, node.Url))
					}
					node.down = false
					node.syncing = false
					node.downSince = time.Unix(0, 0)
					cc.noNodes = false
				}(node)
			}
		}
	}
} 