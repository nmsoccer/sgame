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
然后修改/usr/local/bin/redis.conf新增密码requirepass cbuju 用作sgame使用redis的连接密码;修改port 6698作为监听端口 然后cd /usr/local/bin; ./redis-server ./redis.conf & 拉起即可  

#### 必需库
* **PROTOBUF-C**  
这里用手动安装来说明.
  * 下载安装  
  进入https://github.com/protocolbuffers/protobuf-go 下载protobuf-go-master.zip, 然后拷贝到GOPATH/src: cp protobuf-go-master.zip $GOPATH/src/google.golang.org/; cd $GOPATH/src/google.golang.org/; 解压并改名解压后的目录为protobuf: unzip protobuf-go-master.zip; mv protobuf-go-master/ protobuf/
  * 生成protoc-gen-go工具  
  进入$GOPATH/src 然后go build google.golang.org/protobuf/cmd/protoc-gen-go/ 顺利的话会生成protoc-gen-go二进制文件 然后mv protoc-gen-go /usr/local/bin   
  * 安装proto库  
  进入$GOPATH/src 然后go install google.golang.org/protobuf/proto/ 安装协议解析库
  * 完成   
  进入任意目录执行protoc --go_out=. xxx.proto即可在本目录生成xxx.pb.go文件（这里最好遵循proto3语法）
  
  * 补充安装github.com/golang/protobuf/proto  
  由于生成的xx.pb.go文件总会引用github.com/golang/protobuf/proto库，所以我们一般还需要额外安装这个玩意儿. 进入https://github.com/golang/protobuf页面，下载protobuf-master.zip到本地.
  然后mkdir -p $GOPATH/src/github.com/golang/目录. cp protobuf-master.zip $GOPATH/src/github.com/golang/. 解压并重命名cd $GOPATH/src/github.com/golang/; unzip protobuf-master.zip; mv protobuf-master/ protobuf/; 安装 cd $GOPATH/src; go install github.com/golang/protobuf/proto
  
* **REDIGO**    
redigo是go封装访问redis的支持库，这里仍然以手动安装说明
  * 下载  
  进入https://github.com/gomodule/redigo页面 下载redigo-master.zip到本地
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
    * 进入sgame/目录。 修改bridge.cfg配置，因为我们是本机部署，所以这里修改BRIDGE_USER，BRIDGE_DIR这两个选项使得用户为本机有效用户即可.具体配置项请参考https://github.com/nmsoccer/proc_bridge/wiki/config-detail说明
    * 执行 chmod u+x build.sh; ./build.sh install  
    * 执行 ./manager -i 1 -N sgame 这是一个通信管理工具 执行命令STAT * 可以查看到当前路由的建立情况. 具体使用可以参考https://github.com/nmsoccer/proc_bridge/wiki/manager  

  * 编译进程
    * 进入$GOPATH/src/sgame/servers/spush chmod u+x init.sh build_server.sh
    * 执行./init.sh 初始化设置
    * 执行 ./build_servers.sh -b 编译(也可以进入servers/xx_serv各目录下go build xx_serv.go 手动编译)

  * 发布进程
    * 进入$GOPATH/src/sgame/servers/spush
    * spush是一个分发管理工具，具体使用可以参考https://github.com/nmsoccer/spush 这里也将其集成到了框架内部
    * sgame.json,sgame_shut.json是spush使用的配置文件，我们都是本地部署所以只需要sgame.json，sgame_shut.json文件里的nmsoccer用户名配置成本机有效用户xxx即可
      sed -i "s/nmsoccer/xxx/g" sgame.json; sed -i "s/nmsoccer/xxx/g" sgame.json
    * 发布拉起 
      ./spush -P -f sgame.json 结果如下:
      ```
      ++++++++++++++++++++spush (2020-07-27 17:08:05)++++++++++++++++++++
      .push all procs
      create cfg:4/4 
  
      ----------Push <sgame> Result---------- 
      ok
      .
      [4/4]
      [conn_serv-1]::success 
      [logic_serv-1]::success 
      [db_logic_serv-1]::success 
      [manage_serv-1]::success 

      +++++++++++++++++++++end (2020-07-27 17:08:07)+++++++++++++++++++++
      ```
      说明OK鸟
      
    * 页面监控
    如果拉起进程顺利，我们可以打开页面查看，默认配置是8080 需要用户名及密码 默认配置于spush/tmpl/manage_serv.tmpl:auth配置项。我们选用admin登陆查看：
    ![管理页面](https://github.com/nmsoccer/sgame/blob/master/pic/manage.png)   
    点击具体进程还可以查看到具体的性能指标，这里使用top的输出数据  
    ![详细信息](https://github.com/nmsoccer/sgame/blob/master/pic/man_detail.png)


