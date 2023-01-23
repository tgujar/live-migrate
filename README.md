# live-migrate
An experiment in migrating live containers running on AWS


## About

The project utilizes the live migration facility provided by `runc` container runtime to efficiently migrate containers running on multiple EC2 instances to balance the CPU load. We also use AWS EFS to support fast transfer of frozen container state. 

### Apiserver 
This service runs on each EC2 instance and is responsible to freezing container state or restarting a container. It exposes a REST api so that commands can be issued remotely

### Controller
The controller service is responsible for consuming usage statistics, figure out the most optimal container placement and issue appropriate commands to the Apiserver.

More details about the implementation can be found in our [report](https://github.com/tgujar/live-migrate/blob/main/paper.pdf)
