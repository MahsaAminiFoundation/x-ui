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

config_cronjob_files() {
    echo -e "${yellow}Copying cronjob configs to automatically increase bandwidth weekly${plain}"
    cp /usr/local/x-ui/mahsa_amini_vpn /etc/cron.d/
}

config_telegraf_agent() {
    # change the host name to the subdomain
    subdomain=$(sqlite3 /etc/x-ui/x-ui.db "select value from settings where key='serverName'"| cut -d '.' -f 1)
    [[ ! -z "$subdomain" ]] && hostname $subdomain 

    #install telegraf
    wget -qO- https://repos.influxdata.com/influxdb.key | sudo tee /etc/apt/trusted.gpg.d/influxdb.asc >/dev/null
    source /etc/os-release
    if [[ x"${release}" == x"debian" ]]; then
        echo "deb https://repos.influxdata.com/${ID} ${VERSION_CODENAME} stable" | sudo tee /etc/apt/sources.list.d/influxdb.list
    fi
    sudo apt-get update && sudo apt-get install telegraf

    # make changes to unit file /etc/systemd/system/multi-user.target.wants/telegraf.service
    systemctl stop telegraf
    cat > /lib/systemd/system/telegraf.service << EOF
    [Unit]
    Description=Telegraf
    Documentation=https://github.com/influxdata/telegraf
    After=network-online.target
    Wants=network-online.target

    [Service]
    Type=notify
    EnvironmentFile=-/etc/default/telegraf
    Environment=export INFLUX_TOKEN=eqiMSxfM8VF7Bii0Yqa9bmx0OZSpkbB2hq1uL8NvCZ3urLLj_y40O-hMt7fVBXIMp0Vtz9_h_inoC9vRhWVqNA==
    User=root
    ExecStart=/usr/bin/telegraf --config https://monitoring.mahsaaminivpn.com:8086/api/v2/telegrafs/0bb082ca13341000
    ExecReload=/bin/kill -HUP $MAINPID
    Restart=on-failure
    RestartForceExitStatus=SIGPIPE
    KillMode=control-group

    [Install]
    WantedBy=multi-user.target
EOF

    # reload unit files
    systemctl daemon-reload

    # enable and start service
    systemctl enable telegraf
    systemctl restart telegraf

}

update_x-ui() {
    cd /usr/local/

    last_version=$(curl -Ls "https://api.github.com/repos/MahsaAminiFoundation/x-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ ! -n "$last_version" ]]; then
        echo -e "${red}检测 x-ui 版本失败，可能是超出 Github API 限制，请稍后再试，或手动指定 x-ui 版本安装${plain}"
        exit 1
    fi
    echo -e "Detected x-ui; Latest version：${last_version}，starting insallation"
    wget -N --no-check-certificate -O /usr/local/x-ui-linux-${arch}.tar.gz https://github.com/MahsaAminiFoundation/x-ui/releases/download/${last_version}/x-ui-linux-${arch}.tar.gz
    if [[ $? -ne 0 ]]; then
        echo -e "${red}Failed to download x-ui, please make sure your server can download Github files${plain}"
        exit 1
    fi

    systemctl stop x-ui

    if [[ -e /usr/local/x-ui/ ]]; then
        rm /usr/local/x-ui/ -rf
    fi

    tar zxvf x-ui-linux-${arch}.tar.gz
    rm x-ui-linux-${arch}.tar.gz -f
    /usr/bin/x-ui restart
    config_cronjob_files
    config_telegraf_agent
}

echo -e "${green}Start the update${plain}"
update_x-ui
