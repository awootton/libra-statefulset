#!/bin/bash 

# this is our version of a startup script for libra-
# the files are in mappedconfig/etc but it's read only.

cp -R /mappedconfig/etc  /opt/libra/
chmod +x /opt/libra/etc/startup.sh
cd /opt/libra/etc/ 

name=${POD_NAME}
# eg libra-0

echo pod name is ${name}

ip=${MY_POD_IP}

echo pod ip is ${ip}

N="${name:6:7}"

echo n is ${N}

ipzero=$(getent hosts libra-0.libra | awk '{ print $1 }')
while [ -z "$ipzero" ]; do
    sleep 1
    ipzero=$(getent hosts libra-0.libra | awk '{ print $1 }')
done

myfakeip="10${N}.10${N}.10${N}.10${N}"

echo myfakeip is ${myfakeip}
echo ipzero is ${ipzero}

#sed "s/100.100.100.100/$ipzero/g" ${N}node.config.toml > tmp.txt
#sed "s/$myfakeip/$ip/g" tmp.txt > node.config.toml

for f in *.toml
do
	sed "s/100.100.100.100/$ipzero/g" $f > tmp.txt
    sed "s/$myfakeip/$ip/g" tmp.txt > $f
    echo processed $f
done

/opt/libra/bin/libra-node -f ${N}node.config.toml

