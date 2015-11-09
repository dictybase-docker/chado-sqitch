#!/bin/bash


wait_for_etcd() {
    if [ ${ETCD_CLIENT_SERVICE_HOST+defined} ]
    then
        curl http://${ETCD_CLIENT_SERVICE_HOST}:${ETCD_CLIENT_SERVICE_PORT}/v2/keys/migration/postgresql?wait=true
    else
        echo "did not register with etcd"
    fi
}

register_etcd() {
    if [ ${ETCD_CLIENT_SERVICE_HOST+defined} ]
    then
        curl http://${ETCD_CLIENT_SERVICE_HOST}:${ETCD_CLIENT_SERVICE_PORT}/v2/keys/migration/sqitch -XPUT -d value="complete"
    else
        echo "did not register with etcd"
    fi
}

deploy_chado() {
    cd $PWD
    sqitch config --user engine.pg.client $(which psql)
    if [ ${CHADO_USER+defined} -a ${CHADO_PASS+defined} -a ${CHADO_DB+defined} ]
    then
        if [ ${POSTGRES_SERVICE_HOST+defined} ]
        then
            sqitch target add dictychado db:pg://${CHADO_USER}:${CHADO_PASS}@${POSTGRES_SERVICE_HOST}:${POSTGRES_SERVICE_PORT}/${CHADO_DB}
            sqitch deploy -t dictychado
        else
            echo no postgres host is defined
        fi
    else
        echo does not have any information about the database to deploy
    fi
}

extract_secret() {
    # secrets get mounted in a kube cluster
    if [ -e "/secrets/chadouser" ] 
    then
        CHADO_USER=$(cat /secrets/chadouser)
    fi

    if [ -e "/secrets/chadopass" ] 
    then
        CHADO_PASS=$(cat /secrets/chadopass)
    fi

    if [ -e "/secrets/chadodb" ] 
    then
        CHADO_DB=$(cat /secrets/chadodb)
    fi
}


main() {
    wait_for_etcd
    extract_secret
    deploy_chado
    register_etcd
}


main


