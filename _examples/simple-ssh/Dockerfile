FROM ubuntu:16.04

# Install SSH server and other required packages
RUN apt-get update && apt-get install -y openssh-server sudo vim \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH server
RUN mkdir /var/run/sshd
RUN echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config
RUN echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config

# Create a CTF user with a default password
RUN useradd -m -s /bin/bash ctf-user \
    && echo "ctf-user:ctf-password" | chpasswd \
    && echo "ctf-user ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/ctf-user

# Add the challenge flag in a hidden location
RUN echo "FLAG{SSH_CHALLENGE_FLAG}" > /home/ctf-user/.hidden_flag \
    && chmod 600 /home/ctf-user/.hidden_flag

# Add a welcome message
RUN echo "Welcome to the SSH Challenge! Find the hidden flag." > /etc/motd

# Expose SSH port
EXPOSE 22

# Start SSH service
CMD ["/usr/sbin/sshd", "-D"]
