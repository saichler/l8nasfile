docker pull saichler/nasfile:latest
docker run -d \       
    --name nasfile \
    --restart unless-stopped \
    --network host \
    -v /mnt:/host/mnt \
    -v nasfile-data:/data \
    -e NODE_IP=0.0.0.0 \
    saichler/nasfile:latest
