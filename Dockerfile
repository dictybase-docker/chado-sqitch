FROM perl:5.20 
MAINTAINER Siddhartha Basu<siddhartha-basu@northwestern.edu>

# Install postgres client
RUN apt-get update \ 
    && apt-get -y install postgresql-client curl \
    && rm -rf /var/lib/apt/lists/* \
# Add an user that will be used for install purpose
    && useradd -m -r -s /sbin/nologin -c "Docker image user" caboose \
# Install perl prerequisites
    && cpanm -n DBD::Pg App::Sqitch 


# download the source and extract
RUN cd /home/caboose \  
    && curl -L -O https://github.com/dictyBase/Chado-Sqitch/releases/download/dictychado-1.23/sqitch-dictychado-1.23.tar.gz \
    && tar xvjf sqitch-dictychado-1.23.tar.gz \
    && chown -R caboose /home/caboose 


# Source code folder will be the default landing spot
WORKDIR /home/caboose/sqitch-dictychado-1.23

# Set as default user 
USER caboose

# Startup script
ADD run.sh /home/caboose/sqitch-dictychado-1.23/

# Default command
CMD ["/home/caboose/sqitch-dictychado-1.23/run.sh"]

