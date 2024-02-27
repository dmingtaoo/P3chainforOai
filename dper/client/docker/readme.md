### Step1：运行`dockerfile`生成镜像`p3chain:1.1`

```shell
docker build -t p3chain:1.1 . 
```

### Step2：根据节点数量修改 `compose-generate.sh`

根据实际的的节点数量(不包括`booter`)修改`node_num`

### Step3：运行 `compose-generate.sh` 生成`docker compose `文件

```shell
./compose-generate.sh
```

### Step4：运行`conf/initConfig.go` 生成 `auto` 文件夹
```shell
go run initConfig.go
```

### Step5：运行`docker-compose`

```shell
docker-compose up
```

