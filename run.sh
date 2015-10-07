#!/bin/bash

sqitch config --user core.engine.pg.client `which psql`

# secrets get mounted in a kube cluster
[ -e /secrets/chadouser ] && CHADO_USER=$(cat /etc/chadouser)
[ -e /secrets/chadopass ] && CHADO_PASS=$(cat /etc/chadopass)
[ -e /secrets/chadodb ] && CHADO_DB=$(cat /etc/chadodb)


if [ ${CHADO_USER+defined} -a ${CHADO_PASS+defined} -a ${CHADO_DB+defined} ]
then
    if [ ${POSTGRES_SRV_SERVICE_HOST+defined} ]
    then
        sqitch target add dictychado db:pg://${CHADO_USER}:${CHADO_PASS}@${POSTGRES_SRV_SERVICE_HOST}:${POSTGRES_SRV_SERVICE_PORT}/${CHADO_DB}
        sqitch deploy -t dictychado
    else
        echo no postgres host is defined
    fi
else
    echo does not have any information about the database to deploy
fi




