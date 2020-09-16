/*
=====This is a C API file for parse server-client transport pkg in SGame Framework=====
* more info:https://github.com/nmsoccer/sgame
=======================================================================================*/
//pkg option
#define PKG_OP_NORMAL 0  //normal pkg
#define PKG_OP_ECHO   1  //echo client <-> tcp-serv
#define PKG_OP_VALID   2  //valid connection client-->server[validate] server-->client[enc_key if enc enable]
#define PKG_OP_RSA_NEGO  3  //encrypt by rsa_pub_key to negotiate des key client-->server[encrypted key] server-->client[result]
#define PKG_OP_MAX   32 //max option value

//STAT_KEY
#define CONN_VALID_KEY  "c#s..x*.39&suomeI./().32&show+me_tHe_m0ney$"