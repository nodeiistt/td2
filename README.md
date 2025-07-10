# td2

## Install Docker: 
```
apt update
apt install docker.io -y
systemctl start docker
systemctl enable docker

apt install docker-compose -y

docker-compose --version
```

## Clone Repo

```
git clone https://github.com/nodeiistt/td2.git
cd td2

```

> edit `config.yml`

## Start Docker 
```
docker-compose build --no-cache
docker-compose up -d
```
