#!/bin/bash

set -ex
BASE=$(dirname $0)
cd $BASE

./wait_for_db.sh
echo "19-ffaa:0:1303,[127.0.0.1] server1.xmpp" | sudo tee -a /etc/hosts

echo "GRANT ALL ON jackal.* TO 'jackal'@ IDENTIFIED BY 'password';" | mysql -h mysql_host -u root -ppassword
echo "CREATE DATABASE jackal;" | mysql -h mysql_host -u jackal -ppassword

if [ "$JACKAL_ID" == "server1" ]; then
    echo "Jackal server 1 data"
    cp -r ~/s1_data/* .
    mysql -h mysql_host -D jackal -u jackal -ppassword < server1.sql
else
    echo "Jackal server 2 data"
    cp -r ~/s2_data/* .
    mysql -h mysql_host -D jackal -u jackal -ppassword < server2.sql
fi

./jackal -c example.jackal.yml </dev/null &> jackal.stdout &
