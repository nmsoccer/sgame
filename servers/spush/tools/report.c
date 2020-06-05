#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>

extern int errno;

static int show_help(void)
{
	printf("-p <proc_name> \n");
	printf("-s  <stat> status\n");
	printf("-i   <info> extra info\n");
	printf("-A  <host ip address>\n");
	printf("-P   <port> host port\n");
	return 0;
}


int main(int argc , char **argv)
{
	int opt = 0;
	int status = 0;
	int host_port = 0;
	char host_ip[64] = {0};
	char proc_name[128] = {0};
	char extra_info[128] = {0};
	int i = 0;
	int conn_sock = 0;
	int ret = -1;
	int flags = 0;
	struct sockaddr_in serv_addr;
	char msg[256] = {0};
	char rsp[64] = {0};
	/*获取参数*/
	while((opt = getopt(argc , argv , "p:s:i:A:P:h")) != -1)
	{
		switch(opt)
		{
		case 's':
			status = atoi(optarg);
			break;
		case 'A':
			strncpy(host_ip , optarg , sizeof(host_ip));
			break;
		case 'i':
			if(optarg)
				strncpy(extra_info , optarg , sizeof(extra_info));
			break;
		case 'p':
			strncpy(proc_name , optarg , sizeof(proc_name));
			break;
		case 'P':
			host_port = atoi(optarg);
			break;
		case 'h':
			show_help();
			return 0;
		}
	}

	//check option
	if(strlen(proc_name)<=0 || strlen(host_ip)<=0 || host_port<=0)
	{
		printf("err:neccessary info missed!\n");
		show_help();
		return -1;
	}

	printf("try to send msg to %s:%d proc:%s stat:%d info:%s\n" , host_ip , host_port , proc_name , status , extra_info);
	/***Send Msg*/
	//create sock
	conn_sock = socket(AF_INET , SOCK_DGRAM , 0);
	if(conn_sock < 0)
	{
		printf("create socket failed! err:%s\n" , strerror(errno));
		return -1;
	}

	//non-block
	flags = fcntl(conn_sock , F_GETFL , 0);
	if(flags < 0)
	{
		printf("fcntl failed! err:%s\n" , strerror(errno));
		return -1;
	}

	flags |= O_NONBLOCK;
	ret = fcntl(conn_sock , F_SETFL , flags);
	if(ret < 0)
	{
		printf("fcntl set non-block failed! err:%s\n" , strerror(errno));
		return -1;
	}

	//connect
	serv_addr.sin_family = AF_INET;
	serv_addr.sin_port = htons((unsigned short)host_port);
	ret = inet_aton(host_ip , &serv_addr.sin_addr);
	if(ret < 0)
	{
		printf("converse host_addr failed! ip:%s\n" , host_ip);
		return -1;
	}

	ret = connect(conn_sock , (struct sockaddr *)&serv_addr , sizeof(serv_addr));
	if(ret < 0)
	{
		printf("connect dispatch host failed! addr:%s:%d proc:%s err:%s\n" , host_ip , host_port , proc_name , strerror(errno));
		return -1;
	}


	//write msg
	snprintf(msg , sizeof(msg) , "{\"msg_type\":1,\"msg_proc\":\"%s\",\"msg_result\":%d,\"msg_info\":\"%s\"}" , proc_name , status , extra_info);
	printf("msg is:%s\n" , msg);
	for(i=0; i<3; i++)	//send max 3 times
	{
		do
		{
			memset(rsp , 0 , sizeof(rsp));
			ret = send(conn_sock , msg , strlen(msg) , 0);
			if(ret < 0)
			{
				printf("send msg failed! err:%s\n" , strerror(errno));
				break;
			}

			usleep(50000);
			recv(conn_sock , rsp , sizeof(rsp) , 0);
		}
		while(0);

		//out break
		if(strlen(rsp) > 0)
		{
			printf("[%s]\n" , rsp);
			break;
		}

		sleep(2);
	}


	//exit
	close(conn_sock);
	printf("report finish!\n");
	return 0;
}
