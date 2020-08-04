# sgame
A Simple Game Framework  一个简单的游戏框架  

### Features
* **开发语言** 除在进程通信和日志等模块使用了C作为基础库以外，所有的上层逻辑均由go编写，便于使用go标准库的丰富资源
* **减少依赖** 在开发过程中，除了使用本人之前开发的一些基础日志库和进程通信模块由C实现以外，其余内容都基于go标准库的实现，尽量减少第三方库的使用，努力做到可防可控。在第三方上除了对protobuf和goredis的依赖，没有其他多余的库
* **结构简单** 为了方便说明，使用传统的游戏三层架构，连接层,逻辑层,数据层三种进程. 覆盖了网络连接管理，db操作，表格加载等游戏内常见的逻辑，在后续的开发中可以直接集成已有模块进行调用，减少重复开发。同时提供简单的接口和案例，可以方便增加新的功能进程
* **协议兼容** 在server进程之间使用protobuf进行消息序列化，在server与客户端之间使用json编码传输，尽力提供良好的兼容
* **监控管理** 提供一套发布与管理工具，用于多游戏进程的配置管理与业务进程的监控 
* **未完待续**

### 简单架构  
这是一个简单的架构图，主要以各group里的connect_serv,logic_serv,db_serv三层逻辑为核心，框架功能也以这三层的实现为主。同时为表现扩展性这里部署了两组，它们之间通过disp互相通信.  
更多的架构说明请参考wiki 
![架构](https://github.com/nmsoccer/sgame/blob/master/pic/sgame.png) 


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
然后修改/usr/local/bin/redis.conf新增密码requirepass cbuju 用作sgame使用redis的连接密码;修改port 6698作为监听端口 然后cd /usr/local/bin; ./redis-server ./redis.conf & 拉起即可  

#### 必需库
* **PROTOBUF-GO**  
这里用手动安装来说明.
  * 下载安装  
  进入https://github.com/protocolbuffers/protobuf-go 下载protobuf-go-master.zip, 然后拷贝到GOPATH/src: mkdir -p $GOPATH/src/google.golang.org/; cp protobuf-go-master.zip $GOPATH/src/google.golang.org/; cd $GOPATH/src/google.golang.org/; 解压并改名解压后的目录为protobuf: unzip protobuf-go-master.zip; mv protobuf-go-master/ protobuf/
  * 生成protoc-gen-go工具  
  进入$GOPATH/src 然后go build google.golang.org/protobuf/cmd/protoc-gen-go/ 顺利的话会生成protoc-gen-go二进制文件 然后mv protoc-gen-go /usr/local/bin   
  * 安装proto库  
  进入$GOPATH/src 然后go install google.golang.org/protobuf/proto/ 安装协议解析库
  * 完成   
  进入任意目录执行protoc --go_out=. xxx.proto即可在本目录生成xxx.pb.go文件（这里使用proto3）
  
  * 补充安装github.com/golang/protobuf/proto  
  由于生成的xx.pb.go文件总会引用github.com/golang/protobuf/proto 库，所以我们一般还需要额外安装这个玩意儿. 进入https://github.com/golang/protobuf 页面，下载protobuf-master.zip到本地.
  然后mkdir -p $GOPATH/src/github.com/golang/目录. cp protobuf-master.zip $GOPATH/src/github.com/golang/. 解压并重命名cd $GOPATH/src/github.com/golang/; unzip protobuf-master.zip; mv protobuf-master/ protobuf/; 安装 cd $GOPATH/src; go install github.com/golang/protobuf/proto
  
* **REDIGO**    
redigo是go封装访问redis的支持库，这里仍然以手动安装说明
  * 下载  
  进入https://github.com/gomodule/redigo 页面,下载redigo-master.zip到本地
  * 安装  
  创建目录mkdir -p $GOPATH/src/github.com/gomodule; cp redigo-master.zip $GOPATH/src/github.com/gomodule; 解压并重命名: cd $GOPATH/src/github.com/gomodule; unzip redigo-master.zip; mv redigo-master redigo; 安装: cd $GOPATH/src; go install github.com/gomodule/redigo/redis  
  
* **SXX库**  
sxx库是几个支持库，安装简单且基本无依赖,下面均以手动安装为例  
  * slog  
  一个简单的日志库.https://github.com/nmsoccer/slog. 下载slog-master.zip到本地，解压然后安装:cd slog-master; ./install.sh(需要root权限)
  * stlv
  一个简单的STLV格式打包库. https://github.com/nmsoccer/stlv. 下载stlv-master.zip到本地,解压然后安装:cd stlv-master; ./install.sh(root权限)
  * proc_bridge
  一个进程通信组件，sgame里集成了proc_bridge，这里需要安装支持库即可. https://github.com/nmsoccer/proc_bridge 下载proc_bridge-master.zip到本地，然后解压安装:cd proc_bridge-master/src/library; ./install_lib.sh(root权限)，安装完毕. 更加详细的各种配置请参考https://github.com/nmsoccer/proc_bridge/wiki
  
#### SGAME安装  
这里仍然以手动安装为例
  * 下载安装    
  进入 https://github.com/nmsoccer/sgame; 下载sgame-master.zip到本地; 部署cp sgame-master.zip $GOPATH/src/; cd $GOPATH/src; unzip sgame-master.zip; mv sgame-master sgame 完成
  * 配置通信  
    * 进入 $GOPATH/src/sgame/proc_bridge. (这里的proc_bridge就是上面安装的proc_bridge组件，只是为了方便集成到这个项目里了).然后执行./init.sh初始化一些配置.
    * 进入sgame/目录。 修改bridge.cfg配置，因为我们是本机部署，所以这里修改BRIDGE_USER，BRIDGE_DIR这两个选项使得用户为本机有效用户即可.具体配置项请参考https://github.com/nmsoccer/proc_bridge/wiki/config-detail
    * 执行 chmod u+x build.sh; ./build.sh install  
    * 执行 ./manager -i 1 -N sgame 这是一个通信管理工具 执行命令STAT * 可以查看到当前路由的建立情况. 具体使用可以参考https://github.com/nmsoccer/proc_bridge/wiki/manager  

  * 编译进程
    * 进入$GOPATH/src/sgame/servers/spush chmod u+x init.sh build_server.sh
    * 执行./init.sh 初始化设置
    * 执行 ./build_servers.sh -b 编译(也可以进入servers/xx_serv各目录下go build xx_serv.go 手动编译)

  * 发布进程
    * 进入$GOPATH/src/sgame/servers/spush
    * spush是一个分发管理工具，下载自https://github.com/nmsoccer/spush 这里也将其集成到了框架内部
    * sgame.json,sgame_shut.json是spush使用的配置文件，我们都是本地部署所以只需要sgame.json，sgame_shut.json文件里的nmsoccer用户名配置成本机有效用户xxx即可
      sed -i "s/nmsoccer/xxx/g" sgame.json; sed -i "s/nmsoccer/xxx/g" sgame_shut.json
    * 发布拉起 
      ./spush -P -f sgame.json 结果如下:
      ```
      ++++++++++++++++++++spush (2020-06-27 18:08:05)++++++++++++++++++++
      .push all procs
      create cfg:9/9 
  
      ----------Push <sgame> Result---------- 
      ok
      .
      [9/9]
      [db_logic_serv-2]::success 
      [disp_serv-1]::success 
      [disp_serv-2]::success 
      [manage_serv-1]::success 
      [conn_serv-2]::success 
      [logic_serv-2]::success 
      [db_logic_serv-1]::success 
      [conn_serv-1]::success 
      [logic_serv-1]::success 

      +++++++++++++++++++++end (2020-06-27 18:08:07)+++++++++++++++++++++
      ```
      说明OK鸟  
    
    * 关闭全部进程    
      ./spush -P -f sgame_shut.json即可  
    
    * 单独推送全部进程配置  
      ./spush -P -f sgame.json -O  
    
    * 单独推送全部进程BIN文件  
      ./spush -P -f sgame.json -o
      
    * 单独推送某个进程BIN文件及配置      
      ./spush -p ^logic* -f sgame.json    
      更多spush的使用请参考https://github.com/nmsoccer/spush 
    
    * 页面监控
    如果拉起进程顺利，我们可以打开页面查看，默认配置是8080 需要用户名及密码 默认配置于spush/tmpl/manage_serv.tmpl:auth配置项。我们选用admin登陆查看：
    ![管理页面](https://github.com/nmsoccer/sgame/blob/master/pic/manage.png)   
    点击具体进程还可以查看到具体的性能指标，这里使用top的输出数据  
    ![详细信息](https://github.com/nmsoccer/sgame/blob/master/pic/man_detail.png)  
    也可以在页面进行手动操作具体或全部进程，比如关闭全部或部分进程  
    ![关闭进程ing](https://github.com/nmsoccer/sgame/blob/master/pic/stopping_server.png)  
    ![关闭进程done](https://github.com/nmsoccer/sgame/blob/master/pic/stopped_server.png)   
    

### 代码结构
下面介绍一下框架的代码目录及其功能
```
sgame
|-- client
|-- lib
|   |-- log
|   |-- net
|   `-- proc
|-- pic
|-- proc_bridge
|   |-- carrier
|   |   `-- tools
|   |-- library
|   `-- sgame
|
|-- proto
|   |-- cs
|   `-- ss
|-- servers
|   |-- comm
|   |-- conn_serv
|   |   `-- lib
|   |-- db_serv
|   |   `-- lib
|   |-- logic_serv
|   |   |-- lib
|   |   |-- table
|   |   `-- table_desc
|   |-- disp_serv
|   |   `-- lib
|   |-- manage_serv
|   |   |-- html_tmpl
|   |   `-- lib
|   `-- spush       
|       |-- tmpl
|       `-- tools
|
`-- xls_conv
    `-- xls
```

* **CLIENT**  
这里主要放置了用于测试功能的客户端文件game_cli.go以及用于压测的press.go  

* **LIB**    
这里放置了可以供客户端和服务器进程使用的一些公开基础库，目前主要包括:  
  * log  
  日志库，封装了slog的go包，同时对多协程的日志记录进行了相应处理
  * net   
  网络收发的基本协议格式，定义了CLIENT<->SERVER之间数据传输的基本TLV协议格式  
  * proc  
  服务器进程之间的通信接口，封装了proc_bridge的go包，并提供了相应的API  

* **PIC**  
 这个是图片 不用鸟

* **PROC_BRIDGE**  
  整合了https://github.com/nmsoccer/proc_bridge 里的proc_bridge组件，其中sgame是为该框架特有的目录，里面主要提供了bridge.cfg配置文件用于配置proc_bridge通信  
  
* **PROTO**   
  这里用于定制服务器之间，客户端与服务器之间的协议
  * cs 
  这里制定客户端与服务器之间的通信协议，使用JSON格式。提供了相应的api.go来进行序列&反序列化  
  * ss
  这里制定服务器之间的传输通信协议，使用protobuf3格式。提供了相应的api.go来进行序列&反序列化  
  
* **SERVERS**  
  这里提供了框架的核心业务进程和部署工具  
  * comm  
  服务器进程所依赖的公共库文件，包括用于tcp连接的tcp_serv.go;用于redis通信的redis.go等  
  * conn_serv  
  用于维护客户端连接的接入管理进程，使用tcp协议接入，为每个客户端连接部署一个协程来进行管理。进程main文件为conn_serv.go
    * conn_serv/lib  
  用于保存conn_serv进程使用的库文件  
  
  * logic_serv
  用于负责处理游戏主要逻辑的业务进程。进程main文件为logic_serv.go
    * logic_serv/lib  
  用于保存logic_serv进程使用的库文件  
    * logic_serv/table    
  业务进程经常使用的资源文件，由excel定义转化为json格式存储  
    * logic_serv/table_desc  
  用于描述业务进程所使用资源文件(json格式)的go文件模板  
  
  * db_serv  
  用于负责框架与redis数据库的读写进程。进程main文件为db_serv.go
    * db_serv/lib  
  用于保存db_serv进程使用的库文件
  
  * disp_serv  
  用于不同logic组之间的消息互通  
    * disp_serv/lib  
  用于保存disp_serv进程使用的库文件  
  
    * manage_serv  
  用于负责管理各具体业务进程的管理进程。进程main文件为manage_serv.go
    * manage_serv/lib  
  用于保存manage_serv进程使用的库文件 
    * manage_serv/html_tmpl  
  用于管理页面的模板文件  
  
* **SPUSH**   
之前介绍过的SPUSH文件部署工具，其中tmpl用于各业务进程配置所使用的模板  

* **XLS_CONV**  
用于业务进程所使用的资源转换工具，将excel文件转换为相关的json文件。具体使用可以参考https://github.com/nmsoccer/xlsconv  

### 更多说明  
更多介绍请参阅页面https://github.com/nmsoccer/sgame/wiki
