/*
 * simple_cli.c
 *
 * This is a Demo Client Connect SGame Server Using C.
 * Test Ping Proto.
 *
 * Build:  gcc simple_cli.c ../lib/net/net_pkg.c -o simple_cli
           ./simple_cli <port>
 * More Info:https://github.com/nmsoccer/sgame/wiki/mulit-connect
 * Created on: 2020.8.6
 * Author: nmsoccer
 */
#include <stdio.h>
#include <string.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>


extern int errno;
#define CONN_SERVER_IP "127.0.0.1"

int show_help()
{
	printf("usage: ./simple_cli <port>\n");
	return 0;
}


int main(int argc , char **argv)
{
	struct sockaddr_in remote_addr;
	int conn_fd = -1;
	short port = -1;
	//for send
	char cmd[1024] = {0};
	char pkg_buff[2048] = {0};

	//for recv
	int pkg_len = -1;
	int data_len = -1;
	char recv_buff[2048] = {0};
	unsigned char tag = 0;

    struct timeval tv;
	int ret = -1;
	long curr_ts = 0;
	if(argc != 2)
	{
		show_help();
		return -1;
	}
	//set arg
	port = atoi(argv[1]);
	if(port<=0)
	{
		printf("arg illegal!\n");
		show_help();
	}

	//fd
	conn_fd = socket(AF_INET , SOCK_STREAM , 0);
	if(conn_fd < 0)
	{
		printf("conn_fd failed! err:%s\n" , strerror(errno));
		return -1;
	}

	//connect
	memset(&remote_addr , 0 , sizeof(remote_addr));
	remote_addr.sin_family=AF_INET;  //设置为IP通信
	remote_addr.sin_addr.s_addr=inet_addr(CONN_SERVER_IP); //服务器IP地址
	remote_addr.sin_port=htons(port);
	ret = connect(conn_fd , (struct sockaddr *)&remote_addr , sizeof(struct sockaddr));
	if(ret < 0)
	{
		printf("connect to %s:%d failed! err:%s\n" , CONN_SERVER_IP , port , strerror(errno));
		return -1;
	}
	printf("connect to %s:%d success!\n" , CONN_SERVER_IP , port);


	//Test Ping
	/*
	 * create json request refer sgame/proto/cs/: api.go and ping.proto.go
	*/
    ret = gettimeofday(&tv, NULL);
	curr_ts = tv.tv_sec*1000 + tv.tv_usec/1000;
	snprintf(cmd , sizeof(cmd) , "{\"proto\":1 , \"sub\":{\"ts\":%ld}}" , curr_ts);

	//pack cmd
	pkg_len = PackPkg(pkg_buff, sizeof(pkg_buff) , cmd , strlen(cmd) , 0);
	if(pkg_len <= 0)
	{
		printf("pack pkg failed! ret:%d cmd:%s\n" , pkg_len , cmd);
		return -1;
	}

	//send to server
	ret = send(conn_fd , pkg_buff , pkg_len , 0);
	if(ret < 0)
	{
		printf("send %s failed! err:%s\n" , cmd , strerror(errno));
		return -1;
	}
	printf(">>send cmd:%s success!\n" , cmd);

	//recv
	memset(recv_buff , 0 , sizeof(recv_buff));
	ret = recv(conn_fd , recv_buff , sizeof(recv_buff) , 0);
	if(ret < 0)
	{
		printf("recv failed! ret:%d err:%s\n" , ret , strerror(errno));
		return -1;
	}
	if(ret == 0)
	{
		printf("server closed!\n");
		return -1;
	}

	//unpack
	memset(pkg_buff , 0 , sizeof(pkg_buff));
	tag = UnPackPkg(recv_buff , ret , pkg_buff , sizeof(pkg_buff) , &data_len , &pkg_len);
	if(tag==0xFF)
	{
		printf("unpack failed!\n");
		return -1;
	}
	if(tag == 0xEF)
	{
		printf("pkg_buff not enough!\n");
		return -1;
	}
	if(tag == 0)
	{
	    printf("data not ready!\n");
        return -1;
	}
	printf("<<recv from server: %s data_len:%d pkg_len:%d tag:%d\n" , pkg_buff , data_len , pkg_len , tag);


	close(conn_fd);
	return 0;
}
