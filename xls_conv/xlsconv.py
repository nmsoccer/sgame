#coding=utf-8
'''
created by nmsoccer.
refer https://github.com/nmsoccer/xlsconv 
'''
import sys,getopt;
import xlrd;

print("fuck suomei");
META_SHEET="meta";
PARAM_TYPE_NUMBER = 1; #数字
PARAM_TYPE_STRING = 2; #字符串
PARAM_TYPE_RAW = 3; #原始文本

OUTPUT_PREFIX="output";
#option
VERBOSE = 0;
INPUT_FILE = "";
OUTPUT_FILE = "";
FORMAT = "";

#FORMAT
FORMAT_JSON="json"
FORMAT_XML="xml"


#macro
INDENT = "  " #2 white space
WORKBOOK = None;
SHEET_DICT = {};
MEMORY_SHEET_INFO = [];

def my_print(v):
  if VERBOSE == 1:
    if isinstance(v , unicode):
      print(v.encode('utf-8'));
    else:
      print(v);    

def safe_print(v):
  if isinstance(v , unicode):
    print(v.encode('utf-8'));
  else:
    print(v);  

def conv_str(v):
  if isinstance(v , unicode):
    return v;
  if isinstance(v , str) != True:
    return str(v);  


def open_xls(file_name):
  global WORKBOOK;
  global SHEET_DICT;
  #decode_file_name = file_name.decode('utf-8');
  workbook = xlrd.open_workbook(file_name);
  if workbook == None:
    safe_print('open %s failed!' % file_name);
    return False;  
  safe_print('open input file succcess!');
  WORKBOOK = workbook;
  
  #init each sheet
  sheet_dic = SHEET_DICT;
  sheet_names = workbook.sheet_names();  
  for sheet_name in sheet_names:    
    sheet_dic[sheet_name] = {'valid':0};
  my_print(sheet_dic);
  #check meta sheet
  if sheet_dic.has_key(META_SHEET) == False:
    safe_print("parse failed! no %s found!" % META_SHEET);
    return False;    
  return True;
  
def parse_meta():
  global WORKBOOK;
  global SHEET_DICT;
  #get meta sheet
  sheet = WORKBOOK.sheet_by_name(META_SHEET);
  my_print(sheet);

  #each column define a data sheet
  ncols = sheet.ncols;
  for i in range(ncols):
    each_meta = sheet.col_values(i , 0 , None);
    my_print("meta col:%d" % i);
    my_print(each_meta)
    #line 1 define sheet name
    sheet_name_exp = each_meta[0];
    exp_list = sheet_name_exp.split("=");
    sheet_name = exp_list[0];
    table_name = exp_list[1];
    
    #check sheet_name exist
    if SHEET_DICT.has_key(sheet_name) == False:
      safe_print("sheet %s Not Defined at %s" % (sheet_name , META_SHEET));
      return False;
    
    #parse each param
    sheet_dict = SHEET_DICT[sheet_name];
    sheet_dict['valid'] = 1;
    sheet_dict['table_name'] = table_name;
    sheet_dict['param_dict'] = {};
    param_dict = sheet_dict['param_dict'];

    #meta each row [col_name=[^|#|$]label_name]
    for row in range(len(each_meta)):
      if row == 0:
        continue;
      if each_meta[row] == '':
        continue;
      param_exp_list = each_meta[row].split("=");
      param_dict[param_exp_list[0]] = [];
      my_list = param_dict[param_exp_list[0]];
      #check param-type
      param_name = param_exp_list[1];
      prefix = param_name[0];
      if prefix == '#':
        my_list.append(PARAM_TYPE_NUMBER);
      elif prefix == '$':
        my_list.append(PARAM_TYPE_STRING);
      elif prefix == '^':
        my_list.append(PARAM_TYPE_RAW);      
      else:
        safe_print("parse_meta failed! illegal param_name %s in %s" % (param_name , META_SHEET));
        return False;
      my_list.append(param_name[1:]);  
      
  
  #finsh
  safe_print("parse meta finish!");
  my_print(SHEET_DICT);
  return True;

#parse head line of each normal sheet
def parse_head_line(sheet_name , head_line):  
  #print(head_line)
  sheet_dict = SHEET_DICT[sheet_name];
  if sheet_dict.has_key('param_dict') == False:
    safe_print("parse_head_line failed! sheet %s has no param defined!" % sheet_name);
    return None , None;
    
  param_dict = sheet_dict['param_dict']; #param_dict={'col_name':[type , defined_name] , ...}  
  col_name_refer = [];
  invalid_column = {};
  for i in range(len(head_line)):
    if param_dict.has_key(head_line[i]) == False: #invalid column
      invalid_column[i] = 1;
      continue;
    defined_list = param_dict[head_line[i]];
    col_name_refer.append(defined_list);  
  return col_name_refer , invalid_column;


      
#parse normal sheet
def parse_sheet(sheet_name):
  sheet = WORKBOOK.sheet_by_name(sheet_name);
  my_print(sheet);
  #line 1 construct refer
  col_name_refer , invalid_column = parse_head_line(sheet_name , sheet.row_values(0 , 0 , None));
  if col_name_refer == None:
    safe_print("parse_sheet %s failed! col_name_refer None!" % sheet_name);
    return None;
  my_print(col_name_refer);
  my_print(invalid_column);
  
  
  #parse each row
  sheet_info = {};
  sheet_info['head_refer'] = col_name_refer;
  sheet_info['content'] = [];
  sheet_content = sheet_info['content'];
  nrows = sheet.nrows;
  for i in range(nrows):
    if i == 0: #head line not dd
      continue;
    row_values = sheet.row_values(i , 0 , None);
    valid_values = [];
    for col in range(len(row_values)):
      if invalid_column.has_key(col) == False: #filt invalid column
        valid_values.append(row_values[col]);
        
    sheet_content.append(valid_values);
  safe_print("parse_sheet finish!");  
  return sheet_info;
  
  
  

#parse normal sheets
def parse_sheets():
  global MEMORY_SHEET_INFO;
  for sheet_name in SHEET_DICT:
    if sheet_name == META_SHEET:
      continue;
    if SHEET_DICT[sheet_name]['valid'] != 1: #no valid sheet
      continue;    
    sheet_info = parse_sheet(sheet_name);
    if sheet_info == None:
      return False;
    sheet_info['table_name'] = SHEET_DICT[sheet_name]['table_name'];
    MEMORY_SHEET_INFO.append(sheet_info);
  my_print("parse_sheets finish!");  
  return True;

'''
MEMORY_SHEET_INFO=[sheet_info1 , sheet_info2,....]

sheet_info包含了解析了过的每个sheet的数据，后面的处理均于表格无关。可以将数据打印为自己想要的格式
sheet_info = 
{
'head_refer':[[type , label_name] , [type , label_name] , ...], 
 每个[type,label]以列排序，每个下标位置代表列坐标.
 type:参考PARAM_TYPE_XX,label_name:解析进目标文件的列对应的字段名
'content':[[col1,col2,....] , [col1,col2,col3,],...]
 每个[col1,col2,....]为每行数据 从第二行到最后一行(首行定义列名)。内部的col1,col2,均按列有序代表不同列的值
'table_name':每个sheet在meta中被定义的结构名 
}
'''
def dump_memory_sheets():
  #my_print(MEMORY_SHEET_INFO);    
  if FORMAT == FORMAT_JSON:
    dump_format_json();
    return;    
  if FORMAT == FORMAT_XML:
    dump_format_xml();

####################DUMP JSON START#############################
def dump_format_json():
  #open output
  file = open(OUTPUT_FILE , "w+");
  if file == None:
    safe_print("dump_format_json failed! open output file:%s failed!" % OUTPUT_FILE);
    return;    
  
  #print main body
  file.write("{\n");
  
  #each sheet  
  indent = INDENT;
  for i in range(len(MEMORY_SHEET_INFO)):
    sheet_info = MEMORY_SHEET_INFO[i];
    safe_print("dump sheet '%s' " % sheet_info['table_name']);
    my_print(sheet_info);
    
    #print table
    ss = indent + '"' + sheet_info['table_name'] + '":\n' + indent + "{\n";
    file.write(ss);
    dump_json_base_table(file , sheet_info , indent);
    ss = indent + '}';
    
    #if last
    if i != len(MEMORY_SHEET_INFO)-1:
      ss = ss + ",";
    ss = ss + "\n";      
    file.write(ss);
    
  file.write("}");
  file.close();

def dump_json_base_table(file , sheet_info , parent_indent):
  indent = INDENT + parent_indent;
  content = sheet_info['content'];
    
  #print count
  ss = indent + '"count": ' + str(len(content)) + ",\n";
  file.write(ss);

  #print res
  ss = indent + '"res":\n' + indent + "[\n"
  file.write(ss);
  
  dump_json_content(file , sheet_info , indent);
  
  ss = indent + "]\n";
  file.write(ss);

def dump_json_content(file , sheet_info , parent_indent):
  indent = INDENT + parent_indent;
  content = sheet_info['content']; #like  [[10102.0, u'\u5934\u76d4', 100.0, 101.0], [10203.0, u'\u5e3d\u5b50', 200.0, 102.0], ...]
  head_refer = sheet_info['head_refer']; #like [[1, u'id'], [2, u'name'], [1, u'price'], [1, u'type'] , ...]
  for i in range(len(content)):
    #dump each row
    row = content[i];  
    ss = indent + "{";
    for col in range(len(row)):
      #print col label
      refer = head_refer[col];
      ss = ss + '"' + refer[1] + '":';
      #print col value
      if refer[0] == PARAM_TYPE_NUMBER:
        if row[col] != '':
          ss = ss + str(int(row[col]));
        else: #empty
          ss = ss + str(0);        
      elif refer[0] == PARAM_TYPE_STRING:
        if row[col] != '':
          ss = ss + '"' + conv_str(row[col]) + '"';
        else:
          ss = ss + '""';        
      else:
        if row[col] != '':      
          ss = ss + conv_str(row[col]);
          #ss = ss + '""'; empty raw column value need set explicit
          #need set explicit          
           
      #check last line
      if col != len(row)-1:
        ss = ss + ", ";     
    
    ss = ss + "}";
    if i != len(content)-1:
      ss = ss + ",";
    ss = ss + "\n";
    #ss = ss.encode('utf-8');
    file.write(ss.encode('utf-8'));
####################DUMP JSON END#############################    

####################DUMP XML START#############################
def dump_format_xml():
  #open output
  file = open(OUTPUT_FILE , "w+");
  if file == None:
    safe_print("dump_format_xml failed! open output file:%s failed!" % OUTPUT_FILE);
    return;    
  
  #print head
  file.write('<?xml version="1.0" encoding="UTF-8" ?>\n');
  #print main body
  file.write("<xlsconv>\n");
  
  #each sheet  
  indent = "";
  for i in range(len(MEMORY_SHEET_INFO)):
    sheet_info = MEMORY_SHEET_INFO[i];
    safe_print("dump sheet '%s' " % sheet_info['table_name']);
    my_print(sheet_info);
    
    #print table
    ss = indent + '<' + sheet_info['table_name'] + '>\n';
    file.write(ss);
    dump_xml_base_table(file , sheet_info , indent);
    ss = indent + '</' + sheet_info['table_name'] + '>\n\n';   
    file.write(ss);
    
  file.write("</xlsconv>");
  file.close();
  
  
def dump_xml_base_table(file , sheet_info , parent_indent):
  indent = INDENT + parent_indent;
  content = sheet_info['content'];
    
  #print count
  ss = indent + '<count>' + str(len(content)) + "</count>\n";
  file.write(ss);

  #print res
  ss = indent + '<res>\n';
  file.write(ss);
  
  dump_xml_content(file , sheet_info , indent);
  
  ss = indent + "</res>\n";
  file.write(ss);

def dump_xml_content(file , sheet_info , parent_indent):
  indent = INDENT + parent_indent;
  content = sheet_info['content']; #like  [[10102.0, u'\u5934\u76d4', 100.0, 101.0], [10203.0, u'\u5e3d\u5b50', 200.0, 102.0], ...]
  head_refer = sheet_info['head_refer']; #like [[1, u'id'], [2, u'name'], [1, u'price'], [1, u'type'] , ...]
  for i in range(len(content)):
    #dump each row
    row = content[i];  
    ss = indent + "<entry>\n";
    file.write(ss);
    sub_indent = INDENT + indent;
    for col in range(len(row)):
      #print col label
      refer = head_refer[col];
      ss = sub_indent + '<' + refer[1] + '>';
      #print col value
      if refer[0] == PARAM_TYPE_NUMBER:
        if row[col] != '':
          ss = ss + str(int(row[col]));
        else: #empty
          ss = ss + str(0);        
      elif refer[0] == PARAM_TYPE_STRING:
        if row[col] != '':
          ss = ss + '"' + conv_str(row[col]) + '"';
        else:
          ss = ss + '""';        
      else:
        if row[col] != '':      
          ss = ss + conv_str(row[col]);
          #ss = ss + '""'; empty raw column value need set explicit
          #need set explicit          
      
      #print col
      ss = ss + "</" + refer[1] + '>\n';
      file.write(ss.encode('utf-8'));
      
    ss = indent + "</entry>\n";
    file.write(ss);  
####################DUMP XML END#############################    
    
def show_help():
  print("-h: show help");
  print("-v: verbose output");
  print("-I <input xls file>");
  print("-O [output file name] if not defined,default output-file name is ./output.xx");
  print("-F format of output file. now support: json , xml");
  print("usage:./xlsconv -I xx.xlsx -O yy.json -F json");


def parse_opt(opts):
  global VERBOSE;
  global FORMAT;
  global INPUT_FILE;
  global OUTPUT_FILE;

  #get opt
  for opt , val in opts:
    if opt == "-h":
      return False;
    if opt == "-v":
      VERBOSE = 1;    
    if opt == "-I":
      INPUT_FILE = val;
    if opt == "-O":
      OUTPUT_FILE = val;    
    if opt == "-F":
      FORMAT = val.strip();
      
  #check opt  
  if len(INPUT_FILE) <= 0:
    safe_print("input file not set!");
    return False;
  if (cmp(FORMAT , "json")!=0) and (cmp(FORMAT , "xml")!=0):
    safe_print("convert format:%s not support!" % FORMAT);
    return False;
  if len(OUTPUT_FILE) <= 0:
    if FORMAT == FORMAT_JSON:
      OUTPUT_FILE = OUTPUT_PREFIX + ".json";
    elif FORMAT == FORMAT_XML:    
      OUTPUT_FILE = OUTPUT_PREFIX + ".xml";
  return True; 
  
  
 
#main
def main():  
  #parse opt
  opts , args = getopt.getopt(sys.argv[1:] , "vhI:O:F:");
  if opts == None:
    show_help();
   
  if parse_opt(opts) == False:
    show_help();
    return;    
  

  #open xls
  if open_xls(INPUT_FILE) == False:
    return;

  #parse_meta
  if parse_meta() == False:
    safe_print("parse meata failed!")
    return;
  
  #parse sheets       
  if parse_sheets() == False:
    safe_print("parse sheets failed!");
    return;

  #dump memory info
  #表格数据已经在内存中 只需打印MEMORY_SHEETS即可
  dump_memory_sheets();


main();    