
FROM swr.cn-east-2.myhuaweicloud.com/ief/camerachecker:2.0

RUN chmod +x /usr/bin/ief-camera-checker
ENTRYPOINT ["/bin/bash", "-x", "/root/scripts/start_camera_checker.sh"]

# COPY
# WORKDIR

# ENV DEP_VERSION="0.4.1"
