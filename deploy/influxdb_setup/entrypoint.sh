#!/bin/bash

ping="influx ping --host http://influxdb:8086"

while ! eval $ping; do sleep 3; done

influx setup -f \
    --host http://influxdb:8086 \
    -o ${INFLUXDB_ORG} \
    -b ${INFLUXDB_BUCKET} \
    -u ${ADMIN_USER} \
    -p ${ADMIN_PSWD} \
    -t ${INFLUXDB_TOKEN}
