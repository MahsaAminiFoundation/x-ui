import os
import requests
import json
from urllib.parse import urlparse
from urllib.parse import parse_qs
import base64
from config_dictionary import config_dictionary

MAHSA_LEG_REMARK = "mahsa_leg_vmess" 
ENSURE_NEW_PORT = False
FOREIGN_PROTOCOL = "vless_arash"
URL_SCHEMA = "https"

CONFIG_VLESS_TEMPLATE = """
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

CONFIG_VMESS_TEMPLATE = """
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
      "protocol": "vmess",
      "streamSettings": {
      "network": "tcp",
      "security": "none",
      "tcpSettings": {
         "acceptProxyProtocol": false,
         "header": {
           "type": "http",
           "request": {
             "method": "GET",
             "path": [
               "/"
             ],
             "headers": {}
           },
           "response": {
             "version": "1.1",
             "status": "200",
             "reason": "OK",
             "headers": {}
           }
         }
       }
    },
      "settings": {
        "vnext": [
          {
            "address": "{server_name}",
            "port": {port_number},
            "users": [
              {
                "id": "{uuid}",
                "flow":"xtls-rprx-direct",
                "security": "auto",
                "level": 0,
                "alterId": 0
              }
            ]
          }
        ]
      }
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
    "rules": [
      {
        "inboundTag": [
          "api"
        ],
        "outboundTag": "api",
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
        "outboundTag": "blocked",
        "protocol": [
          "bittorrent"
        ],
        "type": "field"
      }
    ]
  },
  "stats": {}
}

"""
CONFIG_VMESS_WS_TEMPLATE = """
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
      "protocol": "vmess",
      "streamSettings": {
        "network": "ws",
        "security": "none",
        "wsSettings": {
          "path": "/",
          "headers": {}
        }
      },
      "settings": {
        "vnext": [
          {
            "address": "{server_name}",
            "port": {port_number},
            "users": [
              {
                "id": "{uuid}",
                "security": "auto",
                "level": 0,
                "alterId": 0
              }
            ]
          }
        ]
      }
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
    "rules": [
      {
        "inboundTag": [
          "api"
        ],
        "outboundTag": "api",
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
        "outboundTag": "blocked",
        "protocol": [
          "bittorrent"
        ],
        "type": "field"
      }
    ]
  },
  "stats": {}
}
"""

CONFIG_TROJAN_TEMPLATE = """
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
        "protocol": "trojan",
        "settings": {
            "servers": [
                {
                    "address": "{server_name}",
                    "flow": "xtls-rprx-direct",
                    "port": {port_number},
                    "password": "{uuid}"
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
    }      
    ,
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
    "rules": [
      {
        "inboundTag": [
          "api"
        ],
        "outboundTag": "api",
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
        "outboundTag": "blocked",
        "protocol": [
          "bittorrent"
        ],
        "type": "field"
      }
    ]
  },
  "stats": {}
}
"""

CONFIG_VLESS_ARASH_TEMPLATE = """
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
                            "encryption": "none"
                        }
                    ]
                }
            ]
        },
        "streamSettings": {
          "network": "ws",
          "security": "none",
          "wsSettings": {
            "acceptProxyProtocol": false,
            "path": "/graphql",
            "headers": {
              "Host": "{host}"
            }
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

import json
  
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
    

for foreign_server_subdomain in config_dictionary:
    params = config_dictionary[foreign_server_subdomain]
    hosts = params["hosts"]
    foreign_server = params.get('server', "{}.mahsaaminivpn.com:8080".format(foreign_server_subdomain))
        
    if len(hosts) > 0:
        if "port" in params and "uuid" in params and "server" in params:
            server_name, port, uuid, host = params["server"], params["port"], params["uuid"], params["host"]
        else:
            if ENSURE_NEW_PORT:
                get_code_from_server(foreign_server, MAHSA_LEG_REMARK)
                delete_user_from_server(foreign_server, MAHSA_LEG_REMARK)
        
            server_name, port, uuid = get_code_from_server(foreign_server, MAHSA_LEG_REMARK)
            print("config created for mahsa_leg on {}: {}, port: {}, uuid: {}".format(foreign_server, server_name, port, uuid))
            print("port: {}".format(port))
            print("uuid: {}".format(uuid))

        
    for subdomain in hosts:
        print("{} -> {}".format(subdomain, foreign_server_subdomain))
        hostname = "{}.mahsaaminivpn.com:8080".format(subdomain)
        
        config_template = ''
        if FOREIGN_PROTOCOL in ['vless', 'vless_cdn']:
            config_template = CONFIG_VLESS_TEMPLATE
        elif FOREIGN_PROTOCOL == 'vmess':
            config_template = CONFIG_VMESS_TEMPLATE
        elif FOREIGN_PROTOCOL == 'vmess_cdn':
            config_template = CONFIG_VMESS_WS_TEMPLATE
        elif FOREIGN_PROTOCOL == 'trojan':
            config_template = CONFIG_TROJAN_TEMPLATE
        elif FOREIGN_PROTOCOL == 'vless_arash':
            config_template = CONFIG_VLESS_ARASH_TEMPLATE
        else:
            print("unrecognized protocol")
            exit(0)
            
        config_str = config_template.replace(
            "{server_name}", server_name).replace(
            "{port_number}", port).replace(
            "{uuid}", uuid).replace(
            "{host}", host)
        d = {'XrayTemplateConfig': config_str}

        url = "{}://{}/xui/api/update_xray_template".format(URL_SCHEMA, hostname)
        print(url)
        update_resp = requests.post(url, data=d)
        if update_resp.status_code != 200:
            print("failed to update the xrayTemplateConfig for {}: {}".format(hostname, update_resp.text))
            exit(0)
        else:
            resp_json = json.loads(update_resp.text)
            if resp_json["success"] != True:
                print("failed to update the xrayTemplateConfig for {}: {}".format(hostname, update_resp.text))
                print(config_str)
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
    
        delete_user_from_server(hostname, "t2")


    