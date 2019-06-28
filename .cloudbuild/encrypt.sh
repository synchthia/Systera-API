#!/bin/bash
set -e
cd "$(dirname "$0")"/.. && _basement=$PWD

PROJECT_ID="startail-io"
KEYRING_NAME="cloudbuild-secret"
KEYRING_PATH="projects/${PROJECT_ID}/locations/global/keyRings/${KEYRING_NAME}"
KEY_NAME="systera-api-env"
ENV_PATH="$_basement"

if [ ! -e "$ENV_PATH/.env" ]; then
    echo "!! $ENV_PATH/.env does not exists!"
    exit 1
fi

if [ "$(gcloud --project ${PROJECT_ID} kms keys list --location=global --keyring=${KEYRING_PATH} --format="value(name)" | grep -x ${KEYRING_PATH}/cryptoKeys/${KEY_NAME}; echo $?)" = "1" ]; then
    echo "=> Creating keys..."
    gcloud --project ${PROJECT_ID} kms keys create ${KEY_NAME} \
        --location=global \
        --keyring=${KEYRING_NAME} \
        --purpose=encryption
fi

echo ":: Encrypting..."
echo -n "$(cat $ENV_PATH/.env)" | gcloud --project ${PROJECT_ID} kms encrypt \
    --plaintext-file=- \
    --ciphertext-file=$ENV_PATH/.env.enc \
    --location=global \
    --keyring=$KEYRING_NAME \
    --key=$KEY_NAME
