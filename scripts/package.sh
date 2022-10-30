mkdir x-ui
cd x-ui
scp root@<dev-server>:/home/ubuntu/x-ui/main x-ui
cp ~/GoogleDrive/iPronto/MahsaAminiVPN/x-ui/x-ui.s* .
mkdir bin
cd bin
cp ~/GoogleDrive/iPronto/MahsaAminiVPN/x-ui/bin/*.dat .
cd ../..
tar czvf x-ui-linux-amd64.tar.gz x-ui/