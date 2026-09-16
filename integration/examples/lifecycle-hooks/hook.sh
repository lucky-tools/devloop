#!/bin/sh

cat << EOF
Build specification:
    DefaultRepo:    $DEVLOOP_DEFAULT_REPO
    MultiLevelRepo: $DEVLOOP_MULTI_LEVEL_REPO
    RPCPort:        $DEVLOOP_RPC_PORT
    HTTPPort:       $DEVLOOP_HTTP_PORT
    WorkDir:        $DEVLOOP_WORK_DIR
    Image:          $DEVLOOP_IMAGE
    PushImage:      $DEVLOOP_PUSH_IMAGE
    ImageRepo:      $DEVLOOP_IMAGE_REPO
    ImageTag:       $DEVLOOP_IMAGE_TAG
    BuildContext:   $DEVLOOP_BUILD_CONTEXT
EOF

