#!/bin/python
import os
import sys,getopt
import time

#BRIDGE_USER="nmsoccer"
#BRIDGE_DIR="/home/nmsoccer/proc_bridge"
BRIDGE_USER=""
BRIDGE_DIR=""


CFG_LOCAL_DIR="./cfg"
CFG_FILE="carrier.cfg"
PY_VERSION=0

#manager setting
MANAGER_PROC_ID_MIN=1;
MANAGER_PROC_ID_MAX=1000;
MANAGER_NAME_PREFIX="__carrier_manager__";
MANAGER_RECV_SIZE=50;
MANAGER_SEND_SIZE=50;

#parsed info
cfg_options = {};
manager_dict = {};
proc_dict = {}

#get version
#PY_VERSION = sys.version_info.major;
PY_VERSION = int(sys.version[0]);
print("version:%d" % (PY_VERSION));

#version set
if PY_VERSION == 3:
	import subprocess
	import operator;
	commands = subprocess;
	def cmp(a, b):		
		if operator.eq(a,b):
			return 0;
		elif operator.lt(a,b):
			return -1;
		else:
			return 1;
else:
	import commands;
	pass;


def parse_cfg():	
	title = ""
	global manager_dict
	global cfg_options
	global proc_dict
	global BRIDGE_USER
	global BRIDGE_DIR
#	proc_dict = {} #proc info 
	proc_id_check = {};
	line_no = 0;
	
	file_cfg = open('bridge.cfg')	
	while True:
		line_no = line_no + 1;
		raw_line = file_cfg.readline()
		if not raw_line:
			break;		
		raw_line.strip()
		line = raw_line.replace(' ' , '')
		
		if len(line) == 0:
			break;
		if line[0] == '#' or line[0] == '\n':
			continue
		pos = line.find('#'); #remove comments at the end of line
		if pos > 0:
			line = line[0:pos];
		
		
		#get each title
		if line[0] == '[':
			pos = line.find(']')
			title = line[1:pos]
		
		#handle [proc]
		if cmp(title , 'PROC') == 0:
			#print "handle proc"
			#need check default_send_size and default_recv_size
			#if not cfg_options.has_key('default_send_size'): //python3 not support has_key
			if 'default_send_size' not in cfg_options:
				print("Error:line:%d parse [PROC] failed! no [DEFAULT_SEND_SIZE] is defined before!" % (line_no));
				return -1;
			
			if 'default_recv_size' not in cfg_options:
				print("Error:line:%d parse [PROC] failed! no [DEFAULT_RECV_SIZE] is defined before!" % (line_no));
				return -1;
			
			if 'name_space' not in cfg_options or len(cfg_options["name_space"])<=0:
				print("Error:line:%d parse [PROC] failed! no [NAME_SPACE] is defined before!" % (line_no));
				return -1;
			
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()				
				if not raw_line:
					break;
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];				
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue
				if line[0] == '[': #end of proc [/PROC]
					if cmp(line , "[/PROC]") != 0:
						print("Erro:No end [/PROC] found at line:%d" %(line_no));
						return -1;
					break;			
			
				#handle each line
				#print "ready to handle " + line
				
				#get proc_name
				pos = line.find('=')
				if pos <= 0:
					print("Syntax Error:While Get proc_name,line:%d No '=' in %s" % (line_no , line));
					return -1				
				proc_name = line[0:pos]
				#print "proc_name: " + proc_name
				#check duplicate
				#if proc_dict.has_key(proc_name):
				if proc_name in proc_dict:
					print("Error:line:%d proc_name '%s' duplicated! Please Check!" % (line_no , proc_name));
					return -1;
					
				#get rest content
				rest_content = line[pos+1:]
				content = rest_content.split(':')
				lenth = len(content)
				if lenth < 3:
					print("Syntax Error:line:%d Wrong proc info: %s" % (line_no , line));
					return -1
				proc_id = content[0];
				#check proc_id
				if int(proc_id) <= MANAGER_PROC_ID_MAX:
					print("Error:line %d proc_id:%s is below %d!  proc_id between 1 and %d is reserved!! Please reset it!" % (line_no , proc_id,MANAGER_PROC_ID_MAX,MANAGER_PROC_ID_MAX));
					return -1;
					
				if proc_id in proc_id_check:
					print("Error:line:%d proc_id '%s' duplicated! Please Check!" % (line_no , proc_id));
					return -1;
					
				proc_ip = content[1]
				proc_port = content[2]
				send_size = cfg_options["default_send_size"];
				if lenth>=4 and int(content[3])>0:
					send_size = int(content[3]);
				recv_size = cfg_options["default_recv_size"];	
				if lenth>=5 and int(content[4])>0:
					recv_size = int(content[4]);
					
				#print "proc_name:" + proc_name + " proc_id:" + proc_id + " proc_ip:" + proc_ip + \
				# " proc_port:" + proc_port
				 
				#add into to proc_dict 
				proc_info = {}
				proc_info['proc_id'] = proc_id
				proc_info['proc_ip'] = proc_ip
				proc_info['proc_port'] = proc_port
				proc_info['send_size'] = send_size;
				proc_info['recv_size'] = recv_size;
				proc_dict[proc_name] = proc_info
				
				#into check
				proc_id_check[proc_id] = 1;
				
			title = ''
		#handle [BRIDGE_USER]
		elif cmp(title , 'BRIDGE_USER') == 0:
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end [/BRIDGE_USER]
					if cmp(line , "[/BRIDGE_USER]") != 0:
						print("Erro:No end [/BRIDGE_USER] found at line:%d" %(line_no));
						return -1;
					break;
				
				cfg_options["bridge_user"] = line;
				BRIDGE_USER = line;
			title = ''
		#handle [BRIDGE_DIR]
		elif cmp(title , 'BRIDGE_DIR') == 0:
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end [/BRIDGE_DIR]
					if cmp(line , "[/BRIDGE_DIR]") != 0:
						print("Erro:No end [/BRIDGE_DIR] found at line:%d" %(line_no));
						return -1;
					break;
				
				cfg_options["bridge_dir"] = line;
				BRIDGE_DIR = line;
			title = ''
		#handle [NAME_SPACE]
		elif cmp(title , 'NAME_SPACE') == 0:
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end [/NAME_SPACE]
					if cmp(line , "[/NAME_SPACE]") != 0:
						print("Erro:No end [/NAME_SPACE] found at line:%d" %(line_no));
						return -1;
					break;
				
				cfg_options["name_space"] = line;
			title = ''
		#handle [DEFAULT_SEND_SIZE]
		elif cmp(title , 'DEFAULT_SEND_SIZE') == 0:			
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end [/DEFAULT_SEND_SIZE]
					if cmp(line , "[/DEFAULT_SEND_SIZE]") != 0:
						print("Erro:No end [/DEFAULT_SEND_SIZE] found at line:%d" %(line_no));
						return -1;
					break;
				
				cfg_options["default_send_size"] = int(line);				
			title = ''
		#handle [DEFAULT_RECV_SIZE]
		elif cmp(title , 'DEFAULT_RECV_SIZE') == 0:
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end [/DEFAULT_RECV_SIZE]
					if cmp(line , "[/DEFAULT_RECV_SIZE]") != 0:
						print("Erro:No end [/DEFAULT_RECV_SIZE] found at line:%d" %(line_no));
						return -1;
					break;
				
				cfg_options["default_recv_size"] = int(line);				
			title = ''
		#hanlde [MANAGER_ADDR]
		elif cmp(title , 'MANAGER_ADDR') == 0:
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];
								
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end [/MANAGER_ADDR]
					if cmp(line , "[/MANAGER_ADDR]") != 0:
						print("Erro:No end [/MANAGER_ADDR] found at line:%d" %(line_no));
						return -1;
					break;
					
				#handle
				if parse_manager_addr(line, line_no) <= 0:
					print("Error: No Manager is Set! at line:%d" % (line_no));
					return -1;
				else:
					pass;
			
			title = '';
		#handle [channel]
		elif cmp(title , 'CHANNEL') == 0:
			#print "handle channel"
			#check manager
			if len(manager_dict) <= 0:
				print("Erro:At line:%d, [MANAGER_ADDR] should be configured before [CHANNEL]!" % (line_no));
				return -1;
				
			while True:
				line_no = line_no + 1;
				raw_line = file_cfg.readline()
				if not raw_line:
					break;				
				line = raw_line.replace(' ' , '')
				line = line.replace('\r' , '') #delete windows ctrl
				line = line.replace('\n' , '')
				pos = line.find('#'); #remove comments at the end of line
				if pos > 0:
					line = line[0:pos];				
				
				#check condition
				if len(line) == 0:
					continue;
				if line[0] == '#':
					continue					
				if line[0] == '[': #end of channel [/CHANNEL]
					if cmp(line , "[/CHANNEL]") != 0:
						print("Erro:No end [/CHANNEL] found at line:%d" %(line_no));
						return -1;
					break;

				#handle each line				
				#get proc_name
				pos = line.find(':')
				if pos <= 0:
					print("Syntax warning:line:%d While Get proc_name,No ':' in %s " % (line_no , line));
					continue
				#proc_name = line[0:pos]
				names = line[0:pos]
				proc_names = [];
				if '[' in names:
					proc_names = parse_pattern(names);
				else:
					proc_names.append(names);
				
				for proc_name in proc_names:
					if proc_name not in proc_dict:
						print("Error:line:%d Can not find proc_name:%s " % (line_no , proc_name))
						return -1												
					#print "proc_name: " + proc_name
				
					#get proc_info
					proc_info = proc_dict[proc_name]
					if "target_list" in proc_info:
						print("Error:line:%d proc '%s' is already configured before!" % (line_no , proc_name));
						return -1;
				
					#get rest content
					rest_content = line[pos+1:]					
					content = parse_target_list(rest_content);
								
					#add all target to target_list					
					target_list = []
					for target in content:
						if target not in proc_dict:
							print("Error:line:%d Can not find proc_name: %s" % (line_no , target));
							return -1
						target_list.append(target)
				
					#add list to dict
					proc_info['target_list'] = target_list
					proc_dict[proc_name] = proc_info
			
			#if mix_manager_and_proc() < 0:
			#	return -1;
			#if check_addr_duplicate() < 0:
			#	return -1;
			title = ''
	
	if cfg_check() < 0:
		return -1;
	#print "---------proc_dict--------"
	#print manager_dict
	#print proc_dict
	#print cfg_options
	file_cfg.close()
	return 0
	

def cfg_check():
	#check BRIDGE_USER
	if len(BRIDGE_USER) <= 0:
		print("Error:BRIDGE_USER:%s not defined!" % (BRIDGE_USER))
		return -1;
		
	#check BRIDGE_DIR
	if len(BRIDGE_DIR) <= 0:
		print("Error:BRIDGE_DIR not defined!")
		return -1;
	
	#mix
	if mix_manager_and_proc() < 0:
		return -1;
		
	#addr duplicat
	if check_addr_duplicate() < 0:
		return -1;
	
	return 0;
	
def print_dict():
	print("==========cfg_opitons==========");
	for option in cfg_options:
		print("[%s]:%s" % (option , cfg_options[option]));
	print("==========end==========")
	print("==========proc_dict==========")
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		proc_id = proc_info["proc_id"]
		proc_ip = proc_info["proc_ip"]
		proc_port = proc_info["proc_port"]
		
		print("<<<<[%s]>>>>" % (proc_name));
		print("[id]:%s [ip]%s [port]:%s [send_size]:%s [recv_size]:%s" % (proc_id,proc_ip,proc_port,proc_info["send_size"],proc_info["recv_size"]));
		print("[target]:")
		if "target_list" in proc_info:
			target_list = proc_info["target_list"];
			count = 1;
			for target in target_list:
				print (target),
				count = count + 1;
				if((count % 5) == 0):
					print("");
					count = 1;
			print("")
	print("==========end==========")
		
	
def parse_manager_addr(src_line , line_no):
	manager_proc_id = MANAGER_PROC_ID_MIN;
	manager_proc_count = 0;
	#print "parse_manager_addr:%s" % (src_line);
	addr_list = src_line.split(';');
	if len(addr_list) <= 0:
		print("Parse MANAGER_ADDR at line:%d Failed!No Addr Found'" % (line_no));
		return -1;

	for addr in addr_list:
		pos = addr.find(':');
		if pos <= 0:
			continue;

		#ip:
		ip = addr[0:pos];
		#port:
		port = addr[pos+1:];
		#name:	
		name = "%s-%d" % (MANAGER_NAME_PREFIX , manager_proc_count);
		#add info
		manager_dict[name] = {};
		proc_info = manager_dict[name];
		proc_info['proc_id'] = manager_proc_id;
		proc_info['proc_ip'] = ip;
		proc_info['proc_port'] = port;
		proc_info['send_size'] = MANAGER_SEND_SIZE;
		proc_info['recv_size'] = MANAGER_RECV_SIZE;
		
		
		#update
		manager_proc_id = manager_proc_id + 1;
		manager_proc_count = manager_proc_count + 1;
		
	return manager_proc_count;
	
def mix_manager_and_proc():
	tmp_manager_list = {};
	#add each proc target
	
	for manager_name in manager_dict:
		proc_info = manager_dict[manager_name];
		proc_info['target_list'] = [];
		target_list = proc_info['target_list'];
		for proc_name in proc_dict:
			#prevent proc_name and manager_name duplicated!
			if cmp(proc_name , manager_name) == 0:
				print("FATAL ERROR:proc_name:%s is duplicated whith in-defined manager name! Please Change it!" % (proc_name));
				return -1;
			if proc_name not in target_list:
				target_list.append(proc_name);
		tmp_manager_list[manager_name] = 1;
	
	#add manager to diff manager target
	for manager_name in manager_dict:
		proc_info = manager_dict[manager_name];
		target_list = proc_info['target_list'];
		for other_manager in tmp_manager_list:
			if manager_name==other_manager:
				continue;
			target_list.append(other_manager);
	
	#add each manager to proc's target
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if 'target_list' not in proc_info:
			proc_info['target_list'] = [];
		target_list = proc_info['target_list'];
		for manager_name in manager_dict:
			if manager_name not in target_list:
				target_list.append(manager_name);
	
	#add manager to proc_dict
	for manager_name in manager_dict:
		proc_info = manager_dict[manager_name];
		if manager_name not in proc_dict:
			proc_dict[manager_name] = proc_info;
		
	return 0;
	
	
def check_addr_duplicate():
	dup_dict = {};
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		ip = proc_info['proc_ip'];
		port = proc_info['proc_port'];
		addr_key = ip+":"+port;
	
		#check
		if addr_key in dup_dict:
			print("FATAL ERROR! ADDR:%s IS DUPLICATED at [%s] and [%s] Please Check!!!" % (addr_key , proc_name , dup_dict[addr_key]));
			return -1;
		
		#add it
		dup_dict[addr_key] = proc_name;
	
	#print dup_dict;
	return 0;
	
	
def parse_target_list(str_list):
	result = [];
	target_list = str_list.split(',');
	for target in target_list:
		if '[' in target:			
			sub_set = parse_pattern(target);
			if not sub_set:
				print("error! parse target %s failed!" % (target));
				return None;
			for item in sub_set:
				if item not in result:
					result.append(item);
		else:
			if target not in result:
				result.append(target);
	#print result;
	return result;


def parse_pattern(src_content):
	result = [];
	prefix = ""
	rest = ""
	content = src_content;
	
	#get prefix
	pos = content.find('[');
	if pos > 0:
		prefix = content[0:pos]
		content = content[pos+1:];
	else:
		print("err: '%s' has no prefix" % (src_content));
		return None;
	
	#check )
	pos = content.find(']');
	if pos < 0:
		print("err:'%s' has no ']'" % (src_content));
		return None;
	content = content[0:pos];
	
	#parse content
	segments = content.split(';');	
	for seg in segments:
		if '-' not in seg:
			name = prefix + seg;
			if name not in result:
				result.append(name);
			continue;
		#parse xx-xx
		pos = seg.find('-');
		low = int(seg[0:pos]);
		high = int(seg[pos+1:]);
		
		#check low&high
		if low > high:
			print("error:%s left number:%d should be lower than right number:%d" % (seg , low , high));
			return None;
		
		#for-each
		for i in range(low , high+1): #range form [low , high]
			name = "%s%d" % (prefix , i);
			if name not in result:
				result.append(name);		
	#print result;
	return result;

	
def build_one_bridge(proc_name):
	print("<<proc_name: " + proc_name + ">>");
	#build bridge for each proc
	if proc_name not in proc_dict:
		print("build %s failed no info found!" % (proc_name));
		return;
	proc_info = proc_dict[proc_name]
		
	proc_id = proc_info["proc_id"]
	proc_ip = proc_info["proc_ip"]
	proc_port = proc_info["proc_port"]
	send_size = proc_info["send_size"];
	recv_size = proc_info["recv_size"];
	name_space = cfg_options["name_space"];
	bridge_dir = "%s/%s" % (BRIDGE_DIR , name_space);
	#deploy
	cmd = "./deploy_proc.sh %s %s %s %s %s %s 0 %s %s %s" % (proc_name,proc_id,proc_ip,proc_port,BRIDGE_USER,bridge_dir,send_size,recv_size,name_space);
	print(cmd);
	status,out = commands.getstatusoutput(cmd)
	print(out);
	if status != 0:
		return -1
	else:
		return 0;
	
def build_one_bridge_old(proc_name):
	print("<<proc_name: " + proc_name + ">>")
	#build bridge for each proc
	proc_info = proc_dict[proc_name]
		
	proc_id = proc_info["proc_id"]
	proc_ip = proc_info["proc_ip"]
	proc_port = proc_info["proc_port"]
	target_list = []
	if "target_list" not in proc_info: #no target
#		pass
		return 0
	else:
		target_list = proc_info["target_list"]
	
	#stop carrier
	#command = "ssh " + BRIDGE_USER + "@" + proc_ip + " " + "\"killall -s INT carrier\""
	#print command
	#os.system(command)
	
	#mkdir	
	dest_dir = BRIDGE_DIR + "/" + proc_id
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "mkdir -p %s " % dest_dir
	else:
		command = "ssh " + BRIDGE_USER + "@" + proc_ip + " " + "\"mkdir -p \"" + dest_dir
	
	print(command)
	status,out = commands.getstatusoutput(command)
	print(out);
	if status != 0:
		return -1
	
	#cp binary file to dest_dir
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "cp ./creater %s" % dest_dir
	else:
		command = "scp ./creater " + BRIDGE_USER+ "@" + proc_ip + ":" + dest_dir
	print(command);
	status,out = commands.getstatusoutput(command)
	print(out);
	if status != 0:
		return -1
    
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "cp ./deleter %s" % dest_dir
	else:
		command = "scp ./deleter " + BRIDGE_USER+ "@" + proc_ip + ":" + dest_dir
	print(command);
	status,out = commands.getstatusoutput(command)
	print(out);
	if status != 0:
		return -1

	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "cp ./carrier %s" % dest_dir
	else:
		command = "scp ./carrier " + BRIDGE_USER+ "@" + proc_ip + ":" + dest_dir
	print(command);
	status,out = commands.getstatusoutput(command)
	print(out);
	if status != 0:
		return -1
	
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "cp ./tester %s" % dest_dir
	else:
		command = "scp ./tester " + BRIDGE_USER+ "@" + proc_ip + ":" + dest_dir
	print(command);
	status,out = commands.getstatusoutput(command)
	print(out);
	if status != 0:
		return -1
		
	#exe remote command
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command_head = "";
	else:
		command_head = "ssh  " + BRIDGE_USER + "@" + proc_ip + " "
	
	#exe deleter
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "cd " + dest_dir + "; ./deleter -i " + proc_id;
	else:
		command = command_head + "\"cd " + dest_dir + "; ./deleter -i " + proc_id + "\""
	print(command);
	status,out = commands.getstatusoutput(command)
	print(out);
#	if status != 0:
#		return -1

	#exe creater
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "cd " + dest_dir + "; ./creater -i " + proc_id + "; sleep 1";
	else:
		command = command_head + "\"cd " + dest_dir + "; ./creater -i " + proc_id + ";sleep 1 \""
	print(command);
	status,out = commands.getstatusoutput(command)
	print(out);
	
	#exe carrier
	form_target = ""
	for target in target_list:
		target_proc_info = proc_dict[target]
		form_target += "@"
		form_target = form_target + target_proc_info['proc_id'] + "&"
		form_target = form_target + target_proc_info['proc_ip'] + "&"
		form_target = form_target + target_proc_info['proc_port']
	
	#command = "./carrier -i " + proc_id
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = command_head + "cd " + dest_dir + "; ./carrier -i " + proc_id 
		command = command + " -n " + proc_port
		command = command + " -t '" + form_target + "' &"
	else:
		command = command_head + "\"cd " + dest_dir + "; ./carrier -i " + proc_id 
		command = command + " -n " + proc_port
		command = command + " -t '" + form_target + "' 1>/dev/null  2>&1 &\" "
		#command = command + " -t '" + form_target + "'"
	print(command);
	os.system(command)		
	return 0
		
def build_bridge():
	#first start manager
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if int(proc_info["proc_id"]) > int(MANAGER_PROC_ID_MAX):
			continue;
		time.sleep(0.1)
		if build_one_bridge(proc_name) < 0:
			print("++Build " + proc_name + " Failed!++\n");
	#then start normal carrier
	time.sleep(1);
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if int(proc_info["proc_id"]) <= int(MANAGER_PROC_ID_MAX):
			continue;
		time.sleep(0.1)
		if build_one_bridge(proc_name) < 0:
			print("++Build " + proc_name + " Failed!++\n");
			
			
def active_bridge():
	proc_ip_tmp = ""
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name]
		
		proc_ip = proc_info["proc_ip"]
		proc_port = proc_info["proc_port"]
		
		if proc_ip == proc_ip_tmp:
			continue
		proc_ip_tmp = proc_ip
		
		#signal USR1 to carrier
		if proc_ip=="localhost" or proc_ip=="127.0.0.1":
			command = "killall -s USR1 carrier";
		else:
			command = "ssh " + BRIDGE_USER + "@" + proc_ip + " " + "\"killall -s USR1 carrier\""
		print(command);
		os.system(command)
		time.sleep(0.1)
		
def active_one_bridge(proc_name):
	proc_info = proc_dict[proc_name]
	if proc_info == nul:
		return false
	
	proc_ip = proc_info["proc_ip"]
	proc_port = proc_info["proc_port"]
		
	#signal USR1 to carrier
	if proc_ip=="localhost" or proc_ip=="127.0.0.1":
		command = "killall -s USR1 carrier"
	else:
		command = "ssh " + BRIDGE_USER + "@" + proc_ip + " " + "\"killall -s USR1 carrier\""
	print(command);
	os.system(command)
	

def shut_all_bridge():
	#first shut normal carrier
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if int(proc_info["proc_id"]) <= int(MANAGER_PROC_ID_MAX):
			continue;
		time.sleep(0.1)
		shut_one_bridge(proc_name);
		print(" ")
	#then shut manager
	time.sleep(1);
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if int(proc_info["proc_id"]) > int(MANAGER_PROC_ID_MAX):
			continue;
		time.sleep(0.1)
		shut_one_bridge(proc_name);
		print(" ")

def shut_one_bridge(proc_name):
	if proc_name not in proc_dict:
		print("proc <%s> is not exist!" % (proc_name));
		return -1;
	#shut
	proc_info = proc_dict[proc_name];		
	proc_id = proc_info["proc_id"];
	proc_ip = proc_info["proc_ip"];
	proc_port = proc_info["proc_port"];
	name_space = cfg_options["name_space"];
	bridge_dir = "%s/%s" % (BRIDGE_DIR , name_space);
	print("SHUTING <<%s[%s:%s] <%s:%s>>..." % (proc_name , name_space , proc_id , proc_ip , proc_port));
	#shut
	cmd = "./deploy_proc.sh %s %s %s %s %s %s 1 0 0 %s" % (proc_name,proc_id,proc_ip,proc_port,BRIDGE_USER,bridge_dir,name_space)
	print(cmd);
	os.system(cmd);
	print(" ");

	
def push_all_cfg():
	for proc_name in proc_dict:
		push_cfg_file(proc_name);
		print(" ")
	
def push_cfg_file(proc_name):
	if proc_name not in proc_dict:
		print("proc <%s> is not exist!" % (proc_name));
		return -1;
	#shut
	proc_info = proc_dict[proc_name];		
	proc_id = proc_info["proc_id"];
	proc_ip = proc_info["proc_ip"];
	proc_port = proc_info["proc_port"];
	name_space = cfg_options["name_space"];
	bridge_dir = "%s/%s" % (BRIDGE_DIR , name_space);
	print("PUSHING <<%s[%s:%s] <%s:%s>>..." % (proc_name , name_space , proc_id , proc_ip , proc_port));
	#shut
	cmd = "./deploy_proc.sh %s %s %s %s %s %s 2 0 0 %s" % (proc_name,proc_id,proc_ip,proc_port,BRIDGE_USER,bridge_dir,name_space);
	print(cmd);
	os.system(cmd);
	print(" ");

def reload_all_cfg():
	#first reload manager
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if int(proc_info["proc_id"]) > int(MANAGER_PROC_ID_MAX):
			continue;
		time.sleep(0.1)
		reload_cfg_file(proc_name);
	#then reload normal carrier
	time.sleep(1);
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if int(proc_info["proc_id"]) <= int(MANAGER_PROC_ID_MAX):
			continue;
		time.sleep(0.1)
		reload_cfg_file(proc_name);
	
	
def reload_cfg_file(proc_name):
	if proc_name not in proc_dict:
		print("proc <%s> is not exist!" % (proc_name));
		return -1;
	#reload
	proc_info = proc_dict[proc_name];		
	proc_id = proc_info["proc_id"];
	proc_ip = proc_info["proc_ip"];
	proc_port = proc_info["proc_port"];
	name_space = cfg_options["name_space"];
	bridge_dir = "%s/%s" % (BRIDGE_DIR , name_space);
	print("RELOAD CFG <<%s[%s:%s] <%s:%s>>..." % (proc_name , name_space , proc_id , proc_ip , proc_port));
	#exe
	cmd = "./deploy_proc.sh %s %s %s %s %s %s 3 0 0 %s" % (proc_name,proc_id,proc_ip,proc_port,BRIDGE_USER,bridge_dir,name_space);
	print(cmd);
	os.system(cmd);
	print(" ")
	
	
		
def create_cfg():
	#mkdir
	dir_path = CFG_LOCAL_DIR + ""
	for proc_name in proc_dict:
		proc_info = proc_dict[proc_name];
		if proc_info == None:
			print("creat_cfg failed! proc_name:" + proc_name + " has no info!");
			continue;
			
		#create dir
		dir_path = '%s/%s' %(CFG_LOCAL_DIR , proc_name);
		try:
			os.makedirs(dir_path);
		except:
			pass
			#print "%s is already exist" % (dir_path)
		
		#create file
		file_name = dir_path + "/" + CFG_FILE;
		try:
			file = open(file_name , "w+");
		except:
			print("open cfg file:%s failed!" % (file_name));
			exit(0);
		
		#write cfg file
		###Instruction
		file.write("#+-----------------INSTRUCT------------------+\n");
		instruction = "#This File is Created by bridge_build.py, Should not be changed directly! \n\n#proc_name=%s \n#proc_id=%s \n\n#%s \n#https://github.com/nmsoccer/ \n" % (proc_name , proc_info["proc_id"] , time.strftime("%Y-%m-%d %H:%M:%S", time.localtime()))
		file.write(instruction);
		file.write("#+-------------------------------------------+\n\n\n");
		file.flush();
		
		
		#write
		#target head
		file.write("#targetlist\n");
		file.write("[target]\n");
		
		#target value
		if "target_list" in proc_info:
			target_list = proc_info["target_list"];		
			for target in target_list:
				target_info = proc_dict[target];
				if target_info == None:
					print("create_cfg:%s failed. target_proc:%s has no info!" % (file_name , target));
					return;
			
				#print =@target_name&target_proc_id&target_ip_addr&target_port
				value = '=@%s&%s&%s&%s\n' % (target , target_info["proc_id"] , target_info["proc_ip"] , target_info["proc_port"]);
				file.write(value);
		file.flush();
		file.close();
	print("create_cfg success!")
		
def show_help():
	print("usage:./bridge_build.py <OPTION> or python ./bridge_build.py <OPTION>")
	print("OPTION:")
	print("-h: show help")
	print("-a: deploy and build all proc bridges")
	print("-A <proc_name>: deploy and build a bridge of <proc_name>")
	print("-c: create all proc-bridge cfg files")
	print("-C <proc_name>: create proc-bridge cfg file of <proc_name>");
	print("-I: print cfg parsing info");
	print("-p: push all proc-bridge cfg file");
	print("-P <proc_name>: push proc-bridge cfg file of <proc_name>");
	print("-r: reload cfg file by all proc-bridge");
	print("-R <proc_name>: reload cfg file by proc-bridge of <proc_name>");
	print("-s: shut down all proc bridges");
	print("-S <proc_name>:shut down proc-bridge of <proc_name>");

		
def main():
	#parse args	
	#opts,args = getopt.getopt(sys.argv[1:] , "hacCIs:")	
	opts,args = getopt.getopt(sys.argv[1:] , "haA:cC:IpP:rR:sS:")
	if opts == None:
		show_help();
		return;
	
	
	for op,value in opts:
		if op == "-h":
			show_help()
		elif op == "-a":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1
			print("---------[Build Bridge]------------");
			build_bridge() #create shm and run carrier
			print("---------[Complete]------------");
			break;
		elif op == "-A":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1
			print("---------[Build Bridge <%s>]------------" % (value));
			build_one_bridge(value);
			print("---------[Complete]------------");
			break;
		elif op == "-c":
			print("---------[Parse Config file]------------");
			if parse_cfg() < 0:
				return -1;
			print("---------[Create Cfg]------------");
			create_cfg();
			break;
		elif op == "-I":
			print("---------[Parse Config file]------------");
			if parse_cfg() < 0:
				return -1;
			print_dict();
			break;
		elif op == "-p":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1;
			print("---------[Pushing All Cfg]------------");
			push_all_cfg();
			break;
		elif op == "-P":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1
			print("---------[Pushing Cfg <%s>]------------" % (value));
			push_cfg_file(value);
			break;
		elif op == "-r":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1;
			print("---------[Reload Cfg All]------------");
			reload_all_cfg();
			break;
		elif op == "-R":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1
			print("---------[Reload Cfg <%s>]------------" % (value));
			reload_cfg_file(value);
			break;
		elif op == "-s":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1
			print("---------[Shutdown Bridge]------------");
			shut_all_bridge();
			break;
		elif op == "-S":
			#print "---------[Parse Config file]------------"
			if parse_cfg() < 0:
				return -1
			print("---------[Shutdown Bridge <%s>]------------" % (value));
			shut_one_bridge(value);
			break;
		else:
			show_help()
			pass
		
main()
