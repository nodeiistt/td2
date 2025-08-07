const h = 24
const w = 9
const textMax = 115
const textW = 120
let gridH = h
let gridW = w
let gridTextMax = textMax
let gridTextW = textW
let scale = 1
let textColor = "#b0b0b0"

let signColorAlpha = 0.4 // alpha set in loop
let isDark = true

function lightMode() {
    isDark = !isDark
    if (isDark) {
        textColor = "#b0b0b0"
        signColorAlpha = 0.4
        document.body.className = "uk-background-secondary uk-light"
        document.getElementById('canvasDiv').className = "uk-width-expand uk-overflow-auto uk-background-secondary"
        document.getElementById("tableDiv").className = "uk-padding-small uk-text-small uk-background-secondary uk-overflow-auto"
        document.getElementById("legendContainer").className = "uk-nav-center uk-background-secondary uk-padding-remove"
        document.getElementById("logs").style = "background: #080808; height: 300px;"

        return
    }
    textColor = "#3f3f3f"
    signColorAlpha = 0.2
    document.body.className = "uk-background-default uk-text-default"
    document.getElementById('canvasDiv').className = "uk-width-expand uk-overflow-auto uk-background-default"
    document.getElementById("tableDiv").className = "uk-padding-small uk-text-small uk-background-default uk-overflow-auto"
    document.getElementById("legendContainer").className = "uk-nav-center uk-background-default uk-padding-remove"
    document.getElementById("logs").style = "color: #0a0a0a; background: #dddddd; height: 300px;"
}

function fix_dpi(id) {
    let canvas = document.getElementById(id),
        dpi = window.devicePixelRatio;
    gridH = h * dpi.valueOf()
    gridW = w * dpi.valueOf()
    gridTextMax = textMax * dpi.valueOf()
    gridTextW = textW * dpi.valueOf()
    let style = {
        height() {
            return +getComputedStyle(canvas).getPropertyValue('height').slice(0,-2);
        },
        width() {
            return +getComputedStyle(canvas).getPropertyValue('width').slice(0,-2);
        }
    }
    canvas.setAttribute('width', style.width() * dpi);
    canvas.setAttribute('height', style.height() * dpi);
    scale = dpi.valueOf()
}

function legend() {
    const l = document.getElementById("legend")
    l.height = scale * h * 1.2
    const ctx = l.getContext('2d')

    let offset = textW
    let grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#f59e0b');
    grad.addColorStop(0.5, '#fbbf24');
    grad.addColorStop(1, '#f59e0b');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.font = `${scale * 14}px sans-serif`
    ctx.fillStyle = 'grey'
    offset += gridW + gridW/2
    ctx.fillText("proposer",offset, gridH/1.2)

    offset += 65 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#10b981');
    grad.addColorStop(0.5, '#34d399');
    grad.addColorStop(1, '#10b981');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.fillStyle = 'grey'
    offset += gridW + gridW/2
    ctx.fillText("signed",offset, gridH/1.2)

    offset += 50 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#3b82f6');
    grad.addColorStop(0.5, '#60a5fa');
    grad.addColorStop(1, '#3b82f6');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("miss/precommit",offset, gridH/1.2)

    offset += 110 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#8b5cf6');
    grad.addColorStop(0.5, '#a78bfa');
    grad.addColorStop(1, '#8b5cf6');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("miss/prevote", offset, gridH/1.2)

    offset += 90 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, '#ef4444');
    grad.addColorStop(0.5, '#f87171');
    grad.addColorStop(1, '#ef4444');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    ctx.beginPath();
    ctx.moveTo(offset + 1, gridH-2-gridH/2);
    ctx.lineTo(offset + 4 + gridW / 4, gridH-1-gridH/2);
    ctx.closePath();
    ctx.strokeStyle = 'white'
    ctx.stroke();
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("missed", offset, gridH/1.2)

    offset += 59 * scale
    grad = ctx.createLinearGradient(offset, 0, offset+gridW, gridH)
    grad.addColorStop(0, 'rgba(127,127,127,0.3)');
    ctx.fillStyle = grad
    ctx.fillRect(offset, 0, gridW, gridH)
    offset += gridW + gridW/2
    ctx.fillStyle = 'grey'
    ctx.fillText("no data", offset, gridH/1.2)
}

function drawSeries(multiStates) {
    const canvas = document.getElementById("canvas")
    canvas.height = ((12*gridH*multiStates.Status.length)/10) + 30
    fix_dpi("canvas")
    if (canvas.getContext) {
        const ctx = canvas.getContext('2d')
        ctx.font = `${scale * 16}px sans-serif`
        ctx.fillStyle = textColor

        let crossThrough = false
        for (let j = 0; j < multiStates.Status.length; j++) {

            //ctx.fillStyle = 'white'
            ctx.fillStyle = textColor
            ctx.fillText(multiStates.Status[j].name, 5, (j*gridH)+(gridH*2)-6, gridTextMax)

            for (let i = 0; i < multiStates.Status[j].blocks.length; i++) {
                crossThrough = false
                const grad = ctx.createLinearGradient((i*gridW)+gridTextW, (gridH*j), (i * gridW) + gridW +gridTextW, (gridH*j))
                switch (multiStates.Status[j].blocks[i]) {
                    case 4: // proposed
                        grad.addColorStop(0, '#f59e0b');
                        grad.addColorStop(0.5, '#fbbf24');
                        grad.addColorStop(1, '#f59e0b');
                        break
                    case 3: // signed
                        grad.addColorStop(0, '#10b981');
                        grad.addColorStop(0.5, '#34d399');
                        grad.addColorStop(1, '#10b981');
                        break
                    case 2: // precommit not included
                        grad.addColorStop(0, '#3b82f6');
                        grad.addColorStop(0.5, '#60a5fa');
                        grad.addColorStop(1, '#3b82f6');
                        break
                    case 1: // prevote not included
                        grad.addColorStop(0, '#8b5cf6');
                        grad.addColorStop(0.5, '#a78bfa');
                        grad.addColorStop(1, '#8b5cf6');
                        break
                    case 0: // missed
                        grad.addColorStop(0, '#ef4444');
                        grad.addColorStop(0.5, '#f87171');
                        grad.addColorStop(1, '#ef4444');
                        crossThrough = true
                        break
                    default:
                        grad.addColorStop(0, 'rgba(127,127,127,0.3)');
                }
                ctx.clearRect((i*gridW)+gridTextW, gridH+(gridH*j), gridW, gridH)
                ctx.fillStyle = grad
                ctx.fillRect((i*gridW)+gridTextW, gridH+(gridH*j), gridW, gridH)

                // line between rows
                if (i > 0) {
                    ctx.beginPath();
                    ctx.moveTo((i * gridW) - gridW + gridTextW, 2 * gridH + (gridH * j) - 0.5)
                    ctx.lineTo((i * gridW) + gridTextW, 2 * gridH + (gridH * j) - 0.5);
                    ctx.closePath();
                    ctx.strokeStyle = 'rgb(51,51,51)'
                    ctx.strokeWidth = '5px;'
                    ctx.stroke();
                }

                // visual differentiation for missed blocks
                if (crossThrough) {
                    ctx.beginPath();
                    ctx.moveTo((i * gridW) + gridTextW + 1 + gridW / 4, (gridH*j) + (gridH * 2) - gridH / 2);
                    ctx.lineTo((i * gridW) + gridTextW + gridW - (gridW / 4) - 1, (gridH*j) + (gridH * 2) - gridH / 2);
                    ctx.closePath();
                    ctx.strokeStyle = 'white'
                    ctx.stroke();
                }
            }
        }
    }
}