#!/bin/bash

cmd="influx setup -f \
    --host http://influxdb:8086 \
    -o ${INFLUXDB_ORG} \
    -b ${INFLUXDB_BUCKET} \
    -u ${ADMIN_USER} \
    -p ${ADMIN_PSWD} \
    -t ${INFLUXDB_TOKEN}"

while ! eval $cmd; do sleep 3; done
