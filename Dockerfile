# syntax=docker/dockerfile:1
FROM ubuntu
RUN apt update
RUN apt upgrade
RUN apt install -y openssh-server sudo tcpdump iputils-ping telnet systemd vim 
RUN useradd -r -m -d /home/titan -s /bin/bash -g root -G sudo -u 1001 titan 
RUN echo titan:titanpassword | chpasswd
RUN service ssh start
ADD lumerin_amd64 /
ADD lumerinconfig.json /
CMD ["/lumerin"]
