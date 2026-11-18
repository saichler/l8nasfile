docker pull registry.layer-8.dev:44443/nasfile/nasfile:latest
docker run -d \       
    --name nasfile \
    --restart unless-stopped \
    --network host \
    -v /mnt:/host/mnt \
    -v nasfile-data:/data \
    -e NODE_IP=0.0.0.0 \
    registry.layer-8.dev:44443/nasfile/nasfile:latest
