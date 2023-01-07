#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)

# check root
[[ $EUID -ne 0 ]] && echo -e "${red}错误：${plain} 必须使用root用户运行此脚本！\n" && exit 1

# check os
if [[ -f /etc/redhat-release ]]; then
    release="centos"
elif cat /etc/issue | grep -Eqi "debian"; then
    release="debian"
elif cat /etc/issue | grep -Eqi "ubuntu"; then
    release="ubuntu"
elif cat /etc/issue | grep -Eqi "centos|red hat|redhat"; then
    release="centos"
elif cat /proc/version | grep -Eqi "debian"; then
    release="debian"
elif cat /proc/version | grep -Eqi "ubuntu"; then
    release="ubuntu"
elif cat /proc/version | grep -Eqi "centos|red hat|redhat"; then
    release="centos"
else
    echo -e "${red}未检测到系统版本，请联系脚本作者！${plain}\n" && exit 1
fi

arch=$(arch)

if [[ $arch == "x86_64" || $arch == "x64" || $arch == "amd64" ]]; then
    arch="amd64"
elif [[ $arch == "aarch64" || $arch == "arm64" ]]; then
    arch="arm64"
elif [[ $arch == "s390x" ]]; then
    arch="s390x"
else
    arch="amd64"
    echo -e "${red}检测架构失败，使用默认架构: ${arch}${plain}"
fi

echo "Architecture: ${arch}"

if [ $(getconf WORD_BIT) != '32' ] && [ $(getconf LONG_BIT) != '64' ]; then
    echo "本软件不支持 32 位系统(x86)，请使用 64 位系统(x86_64)，如果检测有误，请联系作者"
    exit -1
fi

os_version=""

# os version
if [[ -f /etc/os-release ]]; then
    os_version=$(awk -F'[= ."]' '/VERSION_ID/{print $3}' /etc/os-release)
fi
if [[ -z "$os_version" && -f /etc/lsb-release ]]; then
    os_version=$(awk -F'[= ."]+' '/DISTRIB_RELEASE/{print $2}' /etc/lsb-release)
fi

if [[ x"${release}" == x"centos" ]]; then
    if [[ ${os_version} -le 6 ]]; then
        echo -e "${red}请使用 CentOS 7 或更高版本的系统！${plain}\n" && exit 1
    fi
elif [[ x"${release}" == x"ubuntu" ]]; then
    if [[ ${os_version} -lt 16 ]]; then
        echo -e "${red}请使用 Ubuntu 16 或更高版本的系统！${plain}\n" && exit 1
    fi
elif [[ x"${release}" == x"debian" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        echo -e "${red}请使用 Debian 8 或更高版本的系统！${plain}\n" && exit 1
    fi
fi

install_base() {
    if [[ x"${release}" == x"centos" ]]; then
        yum install wget curl tar sqlite3 -y
    else
        apt install wget curl tar sqlite3 -y
        apt install certbot -y
    fi
}

config_cronjob_files() {
    echo -e "${yellow}Copying cronjob configs to automatically increase bandwidth weekly${plain}"
    cp mahsa_amini_vpn /etc/cron.d/
}

config_ssl() {
    domain_name=$1
        
    certbot certonly --standalone --preferred-challenges http --agree-tos --email mahsa@amini.com -d ${domain_name}
}

#This function will be called when user installed x-ui out of sercurity
config_after_install() {
    config_account=$1
    config_password=$2
    config_port=8080
    domain_name=$3
    server_ip=$4
    cert_file="/etc/letsencrypt/live/${domain_name}/fullchain.pem"
    key_file="/etc/letsencrypt/live/${domain_name}/privkey.pem"
        
    /usr/sbin/ufw disable 
    echo -e "${yellow}Your username will be:${config_account}${plain}"
    echo -e "${yellow}Password will be:${config_password}${plain}"
    echo -e "${yellow}Your panel access port number will be:${config_port}${plain}"
    echo -e "${yellow}Settings confirmed.${plain}"
    /usr/local/x-ui/x-ui setting -username ${config_account} -password ${config_password}
    echo -e "${yellow}Username/Password setting completed${plain}"
    /usr/local/x-ui/x-ui setting -port ${config_port}
    echo -e "${yellow}Panel port setting is completed${plain}"
    /usr/local/x-ui/x-ui setting -port ${config_port}

    echo -e "${yellow}Panel serverName setting will be ${domain_name} completed${plain}"
    /usr/local/x-ui/x-ui setting -serverName ${domain_name}
    echo -e "${yellow}Panel serverIP setting will be ${server_ip} completed${plain}"
    /usr/local/x-ui/x-ui setting -serverIP ${server_ip}

    echo -e "${yellow}Panel public key setting will be ${cert_file} completed${plain}"
    /usr/local/x-ui/x-ui setting -webCertFile ${cert_file}
    echo -e "${yellow}Panel private key setting will be ${key_file} completed${plain}"
    /usr/local/x-ui/x-ui setting -webKeyFile ${key_file}
}

# /etc/letsencrypt/live/devserver.mahsaaminivpn.com/privkey.pem
# /etc/letsencrypt/live/devserver.mahsaaminivpn.com/fullchain.pem

config_cdn_stuff() {
    fake_domain_name=$1
    
    if [[ x"${release}" == x"centos" ]]; then
    else
        apt -y install curl git nginx libnginx-mod-stream python3-certbot-nginx
    fi
    
    echo -e "${yellow}Panel fakeServerName setting will be ${fake_domain_name}${plain}"
    /usr/local/x-ui/x-ui setting -fakeServerName ${fake_domain_name}
}

install_x-ui() {
    systemctl stop x-ui
    cd /usr/local/

    last_version=$(curl -Ls "https://api.github.com/repos/roozbeh/x-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ ! -n "$last_version" ]]; then
        echo -e "${red}检测 x-ui 版本失败，可能是超出 Github API 限制，请稍后再试，或手动指定 x-ui 版本安装${plain}"
        exit 1
    fi
    echo -e "Detected x-ui; Latest version：${last_version}，starting insallation"
    wget -N --no-check-certificate -O /usr/local/x-ui-linux-${arch}.tar.gz https://github.com/roozbeh/x-ui/releases/download/${last_version}/x-ui-linux-${arch}.tar.gz
    if [[ $? -ne 0 ]]; then
        echo -e "${red}Failed to download x-ui, please make sure your server can download Github files${plain}"
        exit 1
    fi

    if [[ -e /usr/local/x-ui/ ]]; then
        rm /usr/local/x-ui/ -rf
    fi

    tar zxvf x-ui-linux-${arch}.tar.gz
    rm x-ui-linux-${arch}.tar.gz -f
    cd x-ui
    chmod +x x-ui bin/xray-linux-${arch}
    cp -f x-ui.service /etc/systemd/system/
    wget --no-check-certificate -O /usr/bin/x-ui https://raw.githubusercontent.com/roozbeh/x-ui/main/x-ui.sh
    chmod +x /usr/local/x-ui/x-ui.sh
    chmod +x /usr/bin/x-ui
    config_ssl $3
    config_after_install $1 $2 $3 $4
    config_cronjob_files
    config_cdn_stuff $5
    #echo -e "如果是全新安装，默认网页端口为 ${green}54321${plain}，用户名和密码默认都是 ${green}admin${plain}"
    #echo -e "请自行确保此端口没有被其他程序占用，${yellow}并且确保 54321 端口已放行${plain}"
    #    echo -e "若想将 54321 修改为其它端口，输入 x-ui 命令进行修改，同样也要确保你修改的端口也是放行的"
    #echo -e ""
    #echo -e "如果是更新面板，则按你之前的方式访问面板"
    #echo -e ""
    systemctl daemon-reload
    systemctl enable x-ui
    systemctl start x-ui
    echo -e "${green}x-ui v${last_version}${plain} 安装完成，面板已启动，"
    echo -e ""
    echo -e "How to use the management script for x-ui: "
    echo -e "----------------------------------------------"
    echo -e "x-ui              - Show admin menu (more functions)"
    echo -e "x-ui start        - Start x-ui Panel"
    echo -e "x-ui stop         - Stop x-ui Panel"
    echo -e "x-ui restart      - Restart x-ui Panel"
    echo -e "x-ui status       - 查看 x-ui 状态"
    echo -e "x-ui enable       - 设置 x-ui 开机自启"
    echo -e "x-ui disable      - 取消 x-ui 开机自启"
    echo -e "x-ui log          - 查看 x-ui 日志"
    echo -e "x-ui v2-ui        - 迁移本机器的 v2-ui 账号数据至 x-ui"
    echo -e "x-ui update       - 更新 x-ui Panel"
    echo -e "x-ui install      - 安装 x-ui Panel"
    echo -e "x-ui uninstall    - 卸载 x-ui Panel"
    echo -e "----------------------------------------------"
}

echo -e "${green}开始安装${plain}"
install_base
# $1 -> username
# $2 -> password
# $3 -> domain name
# $4 -> server ip
install_x-ui $1 $2 $3 $4
