### Create your P3-Chain Project

You can generate a P3-Chain project file by implementing the following steps.

### Modify the configuration file

You should modify the configuration file to complete parameter configuration:

1."settings/booterConfig.json":

2."settings/dperConfig.json":

3."settings/rpcConfig.json":

4."settings/natKind/exIP.txt" : If you plan to deploy the P3-Chain project on the ECS and select Static mode in the json configuration file(NATKind in jsonFile), you need to enter the public IP address of the ECS in this file.

5."settings/natKind/serverAddress.txt": If you plan to deploy the P3-Chain project on the ECS and select Hole mode in the json configuration file(NATKind in jsonFile), you need to enter the public IP address and port number of the third-party secondary ECS in this file.

### Execute script to generate project file

You should execute "build.bat" or "build.sh"(It depends on whether your operating system is Windows or Linux).The script will help you generate a project management directory,it's name is "project".

### Run your P3-Chain project

After completing the above steps, you can run your P3-Chain project by starting the client(p3Chain.exe or p3Chain).