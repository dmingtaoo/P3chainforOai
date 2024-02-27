# 文件相关
iniserver.bat 根据参数启动whisper_client并保存自身的URL以及相应的PUBKEY。
keygenerate.bat 生成一个privatekey（privatekey包含了publickey，也即创建了一对钥匙）。
test.bat 根据所给参数启用automode对whisper_client进行测试。
testinterpreter.bat 使用action解释器模式启动whisper_client。

# whisper_client.exe
当前有两个命令集：
* clientstart: 根据所给参数启动whisper_client。
* keygenerate: 在指定地址创建一个私钥。

# action
whisper_client在启动后需要输入action进行相应操作，action如下：
* SEND|FROM|TO|TTL|MESSAGE：使用whisper协议send一条消息，FROM与TO分别为注册过的私钥与公钥，用以指定这条信息的签名者与接收方。
* SENDFILE|FROM|TO|TTL: 与SEND相同，但是根据一个文件地址来传送一个文件（还未实现该action，但是不难。如果文件大小过大，将一个文件内的信息封装到多个envelope中）。
* WATCH|TO|FUNC: 运行whisper协议，监听以TO为接收方的envelope，然后使用相应的函数处理接收的envelope。
* SAVESELFURL: 将自身节点参数打包成URL并保存到相应地址。
* SELFKEYREGISTER: 将节点自身的公钥与私钥以SELF注册。
* REGISTERPUBKEY: 将默认文件夹中的公钥文件一一读取并注册。
* NEWPRVKEY|NAME: whisper协议一个节点有多个密钥来进行envelope的签名，该action将生成一个新的私钥并以所给Name进行注册。
* SLEEP|SECONDS: 进程挂起多少秒，主要用于automode中等待响应。
* SAVESELFPUBKEY: 将自身节点的公钥信息保存。

# automode
在使用clientstart命令启动时，如给予了-automode参数，whisper_client会自动读取所给地址的actionlist，并逐条执行其中的action，全部运行结束后程序关闭。如未给予-automode参数，则会进入解释器模式，需要手动输入逐条命令，输入q退出程序。



