
FROM docker.io/alpine:latest


ADD ./bin/camera_checker /usr/bin/camera_checker
RUN mv /usr/bin/camera_checker /usr/bin/ief-camera-checker
RUN chmod +x /usr/bin/ief-camera-checker

ADD ./scripts/start_camerachecker.sh /root/scripts/start_camerachecker.sh
RUN chmod +x /root/scripts/start_camerachecker.sh
RUN cat /root/scripts/start_camerachecker.sh

ENTRYPOINT ["/bin/sh", "-x", "/root/scripts/start_camerachecker.sh"]

# COPY
# WORKDIR

# ENV DEP_VERSION="0.4.1"
