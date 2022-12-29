import os
import requests
import json

config_vless_template = """
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
        "protocol": "vless",
        "settings": {
            "vnext": [
                {
                    "address": "{server_name}", 
                    "port": {port_number},
                    "users": [
                        {
                            "id": "{uuid}",
                            "flow": "xtls-rprx-direct",
                            "encryption": "none",
                            "level": 0
                        }
                    ]
                }
            ]
        },
        "streamSettings": {
            "network": "tcp",
            "security": "xtls",
            "xtlsSettings": {
                "serverName": "{server_name}"
            }
        }
    },
    {
      "protocol": "freedom",
      "tag": "direct",
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
        "outboundTag": "direct",
        "type": "field"
      },
      {
        "domain": [
          "ext:iran.dat:ir"
        ],
        "outboundTag": "direct",
        "type": "field"
      },
      {
        "domain": [
          "ext:iran.dat:other"
        ],
        "outboundTag": "direct",
        "type": "field"
      }
    ]
  },
  "stats": {}
}
"""

URL_SCHEMA="http"
config_dictionary = {
    "localhost:54321": {
        "server_name": "HOST",
        "uuid": "UUID",
        "port": "PORT"
    }
}

for hostname in config_dictionary:
    params = config_dictionary[hostname]
    
    config_str = config_vless_template.replace(
        "{server_name}", params["server_name"]).replace(
        "{port_number}", params["port"]).replace(
        "{uuid}", params["uuid"])
    d = {'XrayTemplateConfig': config_str}

    update_resp = requests.post("{}://{}/xui/api/update_xray_template".format(URL_SCHEMA, hostname), data=d)
    if update_resp.status_code != 200:
        print("failed to update the xrayTemplateConfig for {}: {}".format(hostname, update_resp.text))
        exit(0)
    else:
        resp_json = json.loads(update_resp.text)
        if resp_json["success"] != True:
            print("failed to update the xrayTemplateConfig for {}: {}".format(hostname, update_resp.text))
            exit(0)
            
        print(update_resp.text)
        print("json template is updated on {}".format(hostname))
    
    add_user_data = {
        'total': '1',
        'remark': 't2',
        'protocol': 'vmess',
        'port': '129'
    }
    add_user_resp = requests.post("{}://{}/xui/api/add_user".format(URL_SCHEMA, hostname), data=add_user_data)
    if update_resp.status_code != 200:
        print("failed add user to {}".format(hostname))
        exit(0)
    print(add_user_resp.text)
    
    delete_user_data = {
        'remark': 't2'
    }
    del_user_resp = requests.post("{}://{}/xui/api/delete_user".format(URL_SCHEMA, hostname), data=delete_user_data)
    if del_user_resp.status_code != 200:
        print("failed delete user on {}".format(hostname))
        exit(0)
    print(del_user_resp.text)
