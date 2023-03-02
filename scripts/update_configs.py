import os
import requests
import json
from urllib.parse import urlparse
from urllib.parse import parse_qs
import base64
from config_dictionary import config_dictionary
import json

URL_SCHEMA = "https"
JUST_RESTART = true
CONFIG_CDN_TEMPLATE = """
{
  "api": {
    "services": [
      "HandlerService",
      "LoggerService",
      "StatsService"
    ],
    "tag": "api"
  },
  "inbounds": [
    {
      "listen": "127.0.0.1",
      "port": 62789,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      },
      "tag": "api"
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "settings": {}
    },
    {
      "protocol": "blackhole",
      "settings": {},
      "tag": "blocked"
    }
  ],
  "policy": {
    "system": {
      "statsInboundDownlink": true,
      "statsInboundUplink": true
    }
  },
  "routing": {
    "domainStrategy": "AsIs",
    "rules": [
      {
        "inboundTag": [
          "api"
        ],
        "outboundTag": "api",
        "type": "field"
      },
      {
        "domain": [
          "geosite:category-ads-all"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "protocol": [
          "bittorrent"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "ip": [
          "geoip:private"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "domain": [
          "ext:iran.dat:ads"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "domain": [
          "geosite:category-porn"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "ip": [
          "geoip:ir"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "domain": [
          "ext:iran.dat:ir"
        ],
        "outboundTag": "blocked",
        "type": "field"
      },
      {
        "domain": [
          "ext:iran.dat:other"
        ],
        "outboundTag": "blocked",
        "type": "field"
      }
    ]
  },
  "stats": {}
}
"""

def get_code_from_server(server_address, remark_name):
    url = "https://{}/xui/api/add_user".format(server_address)
    myobj = {
        "total": "0",
        "remark": remark_name,
        "protocol": FOREIGN_PROTOCOL}
    resp = requests.post(url,  myobj)
    if resp.status_code != 200:
        print("Request failed!")
        print(url)
        print(resp.text)
        exit(0)
    
    resp_obj = json.loads(resp.text)
    if resp_obj['success'] != True:
        print("Request success is false!")
        exit(0)
    
    print(f"response for {server_address} is {resp_obj['msg']}")
    
    if FOREIGN_PROTOCOL in ['vless', 'vless_cdn']:
        parsed_url = urlparse(resp_obj['msg'])
        uuid = parsed_url.netloc.split('@')[0]
        host_port = parsed_url.netloc.split('@')[1]
        address = host_port.split(":")[0]
        port = host_port.split(":")[1]
    elif FOREIGN_PROTOCOL in ['vmess', 'vmess_cdn']:
        vmess_code = resp_obj['msg']
        json_decoded = base64.b64decode(vmess_code[8:])
        vmess_obj = json.loads(json_decoded)
    
        address = vmess_obj["add"]
        port = str(vmess_obj["port"])
        uuid = vmess_obj["id"]
    elif FOREIGN_PROTOCOL == 'trojan':
        parsed_url = urlparse(resp_obj['msg'])
        uuid = parsed_url.netloc.split('@')[0]
        host_port = parsed_url.netloc.split('@')[1]
        address = host_port.split(":")[0]
        port = host_port.split(":")[1]
    elif FOREIGN_PROTOCOL == 'vless_arash':
        parsed_url = urlparse(resp_obj['msg'])
        uuid = parsed_url.netloc.split('@')[0]
        host_port = parsed_url.netloc.split('@')[1]
        address = host_port.split(":")[0]
        port = host_port.split(":")[1]
        host = parse_qs(parsed_url.query)['host']
        
        
    return address, port, uuid
  
def delete_user_from_server(hostname, remark):
    print("deleting {} from {}".format(remark, hostname))
    delete_user_data = {
        'remark': remark
    }
    del_user_resp = requests.post("{}://{}/xui/api/delete_user".format(URL_SCHEMA, hostname), data=delete_user_data)
    if del_user_resp.status_code != 200:
        print("failed delete user on {}".format(hostname))
        exit(0)
    print(del_user_resp.text)

def add_user_from_server(hostname, remark):
    add_user_data = {
        'total': '1',
        'remark': remark,
        'protocol': 'vmess',
        'port': '129'
    }
    add_user_resp = requests.post("{}://{}/xui/api/add_user".format(URL_SCHEMA, hostname), data=add_user_data)
    if add_user_resp.status_code != 200:
        print("failed add user to {}".format(hostname))
        exit(0)
    print(add_user_resp.text)
    

foreign_server_subdomain = "cdn_servers"
params = config_dictionary[foreign_server_subdomain]
hosts = params["hosts"]
foreign_server = params.get('server', "{}.mahsaaminivpn.com:8080".format(foreign_server_subdomain))
        
if len(hosts) == 0:
    exit(0)
        
for subdomain in hosts:
    print("{} -> {}".format(subdomain, foreign_server_subdomain))
    hostname = "{}.mahsaaminivpn.com:8443".format(subdomain)

    if not JUST_RESTART:
        config_str = CONFIG_CDN_TEMPLATE
        d = {'XrayTemplateConfig': config_str}
        url = "{}://{}/xui/api/update_xray_template".format(URL_SCHEMA, hostname)
        print(url)
        update_resp = requests.post(url, data=d)
        if update_resp.status_code != 200:
            print("failed to update the xrayTemplateConfig for {}: {}".format(
                hostname, update_resp.text))
            exit(0)
        else:
            resp_json = json.loads(update_resp.text)
            if resp_json["success"] != True:
                print("failed to update the xrayTemplateConfig for {}: {}".format(
                    hostname, update_resp.text))
                print(config_str)
                exit(0)

            print(update_resp.text)
            print("json template is updated on {}".format(hostname))

    add_user_from_server(hostname, "t2")
    delete_user_from_server(hostname, "t2")

