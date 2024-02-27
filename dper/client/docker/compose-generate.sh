node_num=4
file_name=docker-compose.yaml
touch $file_name

docker rm booter
for ((i=1; i<=node_num; i++)); do
  docker rm node$i
done

echo 'version: "3"' >> $file_name

echo '
services:
  booter:
    image: p3chain:1.1
    stdin_open: true
    working_dir: /home
    volumes:
      - ./../auto/dper_booter1:/home/dper
      - ./perpare.sh:/home/perpare.sh
    container_name: booter
    command: ./perpare.sh
    network_mode: host '>> $file_name

for ((i=1; i<=node_num; i++)); do
  echo "  node$i:
    image: p3chain:1.1
    working_dir: /home
    volumes:
      - ./../auto/dper_dper$i:/home/dper
      - ./perpare.sh:/home/perpare.sh
    container_name: node$i
    command: ./perpare.sh
    depends_on:
      - booter
    network_mode: host " >> $file_name
done