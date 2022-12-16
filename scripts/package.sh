On dev server:
go build -ldflags "-s -w" main.go 


On mac laptop:
cd /Users/roozbeh/GoogleDrive/iPronto/MahsaAminiVPN/x-ui-binary
scp ubuntu@140.82.48.210:/home/ubuntu/x-ui/main x-ui/x-ui
cp ~/GoogleDrive/iPronto/MahsaAminiVPN/x-ui/crontab/mahsa_amini_vpn x-ui/
tar czvf x-ui-linux-amd64.tar.gz x-ui/

mkdir x-ui
cd x-ui
scp root@<dev-server>:/home/ubuntu/x-ui/main x-ui
cp ~/GoogleDrive/iPronto/MahsaAminiVPN/x-ui/x-ui.s* .
cp ~/GoogleDrive/iPronto/MahsaAminiVPN/x-ui/crontab/mahsa_amini_vpn x-ui/
mkdir bin
cd bin
cp ~/GoogleDrive/iPronto/MahsaAminiVPN/x-ui/bin/*.dat .
cd ../..
tar czvf x-ui-linux-amd64.tar.gz x-ui/



#Install iran server
scp /Users/roozbeh/GoogleDrive/iPronto/MahsaAminiVPN/x-ui-binary/x-ui-taghi-linux-amd64.tar.gz  ubuntu@37.32.24.47:.




@tunneling
sysctl net.ipv4.ip_forward=1
iptables -t nat -A PREROUTING -p tcp --dport 22 -j DNAT --to-destination IRAN_IP
iptables -t nat -A PREROUTING -j DNAT --to-destination EU_IP
iptables -t nat -A POSTROUTING -j MASQUERADE