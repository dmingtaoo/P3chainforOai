#!/bin/bash

mulMachineUser_1="oem"
mulMachineIP_1="10.129.68.44"
mulMachineSrc_1="/home/oem/github/MyDP_Chain/dper/client/auto/10.129.68.44"
mulMachineDst_1="/home/oem/github/MyDP_Chain/dper/client/auto/10.129.68.44"

mulMachineUser_2="jackson"
mulMachineIP_2="10.129.118.55"
mulMachineSrc_2="/home/oem/github/MyDP_Chain/dper/client/auto/10.129.118.55"
mulMachineDst_2="/home/jackson/auto/10.129.118.55"

mulMachineUser_3="zwd"
mulMachineIP_3="10.129.121.202"
mulMachineSrc_3="/home/oem/github/MyDP_Chain/dper/client/auto/10.129.121.202"
mulMachineDst_3="/home/zwd/auto/10.129.121.202"


# ## 关闭之前正在运行的P3-Chain节点程序
# mulMachine_1_Stop="$mulMachineDst_1/stopAll.sh"
# mulMachine_2_Stop="$mulMachineDst_2/stopAll.sh"
# mulMachine_3_Stop="$mulMachineDst_3/stopAll.sh"

# bash $mulMachine_1_Stop &
# ssh $mulMachineUser_2@$mulMachineIP_2 "bash $mulMachine_2_Stop" &
# ssh $mulMachineUser_3@$mulMachineIP_3 "bash $mulMachine_3_Stop" &
# sleep 5
# echo "stop nodes done!"

# 复制主机器上的各个IP文件夹到相应的从主机上
ssh $mulMachineUser_2@$mulMachineIP_2 "rm -rf $mulMachineDir_2"
ssh $mulMachineUser_2@$mulMachineIP_2 "bash -c \"sudo -u otherUser bash -c 'mkdir -p $mulMachineDst_2'\""
scp -r $mulMachineSrc_2 $mulMachineUser_2@$mulMachineIP_2:$mulMachineDst_2

ssh $mulMachineUser_3@$mulMachineIP_3 "rm -rf $mulMachineDir_3"
ssh $mulMachineUser_3@$mulMachineIP_3 "bash -c \"sudo -u otherUser bash -c 'mkdir -p $mulMachineDst_3'\""
scp -r $mulMachineSrc_3 $mulMachineUser_3@$mulMachineIP_3:$mulMachineDst_3

# echo "move machine Dir done!"

## 启动本机和远程机器上的P3-Chain节点程序
mulMachine_1_Start="$mulMachineDst_1/startAll.sh"
mulMachine_2_Start="$mulMachineDst_2/startAll.sh"
mulMachine_3_Start="$mulMachineDst_3/startAll.sh"

cd $mulMachineDst_1
bash $mulMachine_1_Start 
sleep 1

ssh $mulMachineUser_2@$mulMachineIP_2 "cd $mulMachineDst_2" 
ssh $mulMachineUser_3@$mulMachineIP_3 "cd $mulMachineDst_3" 

ssh $mulMachineUser_2@$mulMachineIP_2 "bash $mulMachine_2_Start" &
ssh $mulMachineUser_3@$mulMachineIP_3 "bash $mulMachine_3_Start" &
echo "start nodes done!"


