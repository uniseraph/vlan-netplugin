# Omege Vlan Network Plugin

vlan-netplugin 提供了libnetwork的VLAN/VXLAN模式。

## Prerequisite

* 本文档所示示例均在Docker-1.12下完成，请保证Docker版本>=1.12

* 示例所需Docker环境均需要运行在集群模式下，即Docker daemon启动时需要添加

    ```
    --cluster-store=zk://<addr1>
    --cluster-advertise=eth0:2376
    ```

    具体参数含义请参考[Docker daemon help](http://docs.docker.com/engine/reference/commandline/daemon)

* 集群模式Swarm >=1.0.1

## QuickStart


* 编译
```
make build
```


* 制作镜像
```
make build image
```


* 运行（容器方式）
```
    make run  ClUSTER_STORE=zk://localhost:2181 TRUNK_DEV=eth0
```
CLUSTER_STORE 和 TRUNK_DEV有默认值，上述命令相当于
```
    make run
```

* 创建一个vlan段
```
   docker network create -d vlan --gateway=10.230.130.1  --subnet=10.230.130.0/24  --opt VlanId=130 vlan130
```

* 创建一个vlan 容器
```
    docker run -d --net=vlan130 nginx
```
