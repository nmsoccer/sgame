/*
 * simple_cli.java
 *
 * This is a Demo Client Connect SGame Server Using JAVA.
 * Test Ping Proto.
 *
 * Build:  javac simple_cli.java -sourcepath ../lib/net ../lib/net/net_pkg.java -d .
           java simple_cli <port>
 * More Info:https://github.com/nmsoccer/sgame/wiki/mulit-connect
 * Created on: 2020.8.6
 * Author: nmsoccer
 */
import java.net.*;
import java.io.*;
import java.util.*;


public class simple_cli {
    private static String CONN_SERVER_IP = "127.0.0.1";

    public static void main(String[] args) {
        byte[] buff;
        byte[] pkg_buff;
        int pkg_len;
        int data_len;
        int tag;
        int i;
        int port = 0;
        int ret = 0;

        if(args.length != 1) {
            System.out.printf("usage:./simple_cli <port>\n");
            return;
        }
        port = Integer.parseInt(args[0]);
        if(port <= 0) {
            System.out.printf("port:%d illegal! usage:./simple_cli <port>\n" , port);
            return;
        }

        buff = new byte[1024];
        pkg_buff = new byte[1024];
        try{
            //connect
            System.out.println("target:" + CONN_SERVER_IP + "port:" + port);
            Socket client = new Socket(CONN_SERVER_IP, port);
            System.out.println("target_addr:" + client.getRemoteSocketAddress());

            //Test Ping
            /*
             * create json request refer sgame/proto/cs/: api.go and ping.proto.go
            */
            Date date = new Date();
            long curr_ts = date.getTime();
            String cmd = String.format("{\"proto\":1 , \"sub\":{\"ts\":%d}}" , curr_ts);

            //pack cmd
            pkg_len = net_pkg.PackPkg(pkg_buff , cmd.getBytes() , net_pkg.PKG_OP_NORMAL);
            if(pkg_len <= 0){
                System.out.printf("pack failed! pkg_len:%d" , pkg_len);
                return;
            }

            //send
            OutputStream outToServer = client.getOutputStream();
            DataOutputStream out = new DataOutputStream(outToServer);
            out.write(pkg_buff , (int)0 , pkg_len);
            System.out.printf(">>send cmd:%s data_len:%d pkg_len:%d success!\n" , cmd , cmd.length() , pkg_len);

            //recv
            InputStream inFromServer = client.getInputStream();
            DataInputStream in = new DataInputStream(inFromServer);
            ret = in.read(buff);
            if(ret <= 0) {
                System.out.printf("read failed! ret:%d\n" , ret);
                return;
            }

            //unpack
            Arrays.fill(pkg_buff, (byte)0);
            int[] pkg_attr = new int[2];
            tag = net_pkg.UnPackPkg(buff , pkg_buff , pkg_attr);
            if(tag==0xFF){
            	System.out.printf("unpack failed!\n");
            	return;
            }
            if(tag == 0xEF){
            	System.out.printf("pkg_buff not enough!\n");
            	return;
            }
            if(tag == 0){
                System.out.printf("data not ready!\n");
                return;
            }
            data_len = pkg_attr[0];
            pkg_len = pkg_attr[1];
            byte[] result = new byte[data_len];
            System.arraycopy(pkg_buff, 0 , result, 0, data_len);
            String resp = new String(result);
            System.out.printf("<<from server: %s data_len:%d pkg_len:%d tag:%d\n" , resp , data_len , pkg_len , tag);




        } catch(IOException e){
            e.printStackTrace();
        }

    }
}