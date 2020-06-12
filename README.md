# sgame
A Simple Game Framework  一个简单的游戏框架  

### 主要特点
* **开发语言** 除在进程通信和日志等模块使用了C作为基础库以外，所有的上层逻辑均由go编写，便于使用go丰富的资源
* **减少依赖** 在开发过程中，除了使用本人resp下的一些基础C日志和进程通信模块以外，都尽力基于go标准库的实现，尽量减少第三方库的使用. 在第三方上除了对protobuf和goredis的依赖，没有其他多余的库
* **便于扩展** 使用传统的游戏三层架构，connect_server,logic_server,db_server. 覆盖了网络连接，db传输等功能开发，在后续的开发中可以直接集成已有模块来进行网络链接和db调用，减少重复开发
* **协议兼容** 在server进程之间使用protobuf进行消息传递，在server与客户端之间使用json传输，尽力提供良好的兼容


### to be continue
