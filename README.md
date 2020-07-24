# sgame
A Simple Game Framework  一个简单的游戏框架  

### Features
* **开发语言** 除在进程通信和日志等模块使用了C作为基础库以外，所有的上层逻辑均由go编写，便于使用go标准库的丰富资源
* **减少依赖** 在开发过程中，除了使用本人之前开发的一些基础日志库和进程通信模块由C实现以外，其余内容都基于go标准库的实现，尽量减少第三方库的使用，努力做到可防可控。在第三方上除了对protobuf和goredis的依赖，没有其他多余的库
* **结构简单** 为了方便说明，使用传统的游戏三层架构，连接层,逻辑层,数据层三种进程. 覆盖了网络连接管理，db操作，表格加载等游戏内常见的逻辑，在后续的开发中可以直接集成已有模块进行调用，减少重复开发。同时提供简单的接口和案例，可以方便增加新的功能进程
* **协议兼容** 在server进程之间使用protobuf进行消息序列化，在server与客户端之间使用json编码传输，尽力提供良好的兼容
* **监控管理** 提供一套发布与管理工具，用于多游戏进程的配置管理与业务进程的监控 
* **未完待续**

### 安装
#### 基础软件
* **GO**  
下载页面https://golang.google.cn/dl/ 或者 https://golang.org/dl/   
这里下载并使用go 1.14版本，然后tar -C /usr/local -xzf go1.14.6.linux-amd64.tar.gz  修改本地.bashrc 
export PATH=$PATH:/usr/local/go/bin export GOPATH=/home/nmsoccer/go 

* **PROTOBUF**  
下载页面https://github.com/protocolbuffers/protobuf/releases  
这里选择下载protobuf-all-3.11.4.tar.gz.解压到本地后./configure --prefix=/usr/local/protobuf; make; make install  
修改本地.bashrc export PATH=$PATH:/usr/local/protobuf/bin

* **REDIS**  
下载页面https://redis.io/download  
这里选择下载redis-5.0.8.tar.gz. 解压到本地后make 然后拷贝src/redis-cli src/redis-server src/redis.conf 到/usr/local/bin.
然后修改/usr/local/bin/redis.conf新增密码requirepass cbuju 用作sgame使用redis的连接密码


### 进程监控
框架提供了一套简单的进程监控和可视化管理机制，包括了上报协议及管理进程.登陆manage server 配置里的ip:port(这里是localhost:8080)可以打开页面  
这里是个简单展示  
![管理页面](https://github.com/nmsoccer/sgame/blob/master/pic/manage.png)
### to be continue
