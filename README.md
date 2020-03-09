# libra-statefulset
A ~~rust~~ go utility to create a set of Libra validator nodes in a kubernetes namespace. 

Prerequisites: kubectl must be set up to access the target cluster. eg. ```kubectl get nodes``` should return something. Go will be required and rustc and cargo are needed to work with Libra.

TL;DR Just run the Apply function here and it does everything and then you have your own Libra test net. 

See: https://developers.libra.org/docs/run-local-network

See:  https://developers.libra.org/blog/2019/10/29/a-guide-to-running-libra-validators  

See:  https://github.com/libra/libra  ( and do a ```git clone https://github.com/libra/libra.git```)

Step two is to git clone the libra project and cd to the libra directory and run ```docker/validator/build.sh``` and then ```docker/mint/build.sh```
This takes a ~20 minutes each and I had to give docker more cpu and memory to not fail. We'll try to avoid repeating that too much.

Then I label and push them:
```
	docker tag e0e17d4611e0 gcr.io/fair-theater-238820/libra_e2e
	docker tag ec5a821668c4 gcr.io/fair-theater-238820/libra_mint

	docker push gcr.io/fair-theater-238820/libra_e2e
	docker push gcr.io/fair-theater-238820/libra_mint
```
Later they should have version numbers.

Or, you could skip this step use the ones at my gcr which are public. 

Next run the ```Apply()``` function in apply.go. The way I do it is that I run the test (```TestApply```) since I have to do this over and over and over.  The way I brought it up is to first get the stateful set running and keeping libra from running by uncommenting:
```
		command: ["/bin/sh"]
        args: ["-c", "while true; do sleep 10000; done"]
```
in libra-validator and running. Then we can ```kubectl exec -it libra-0 -- bash``` and have a look around. We can check that the volume is mounted at ```/opt/libra/data``` and that the binary is at  ```/opt/libra/bin/libra-node```. Note that we can also see a binary at ```/opt/libra/bin/config-builder``` which claims to be able to generate config for us.

At the root there is a file ```/docker-run.sh``` whose contents are:
```
#!/bin/sh
# Copyright (c) The Libra Core Contributors
# SPDX-License-Identifier: Apache-2.0
set -ex

cd /opt/libra/etc
echo "$NODE_CONFIG" > node.config.toml
echo "$SEED_PEERS" > seed_peers.config.toml
echo "$NETWORK_KEYPAIRS" > network_keypairs.config.toml
echo "$CONSENSUS_KEYPAIR" > consensus_keypair.config.toml
echo "$FULLNODE_KEYPAIRS" > fullnode_keypairs.config.toml
exec /opt/libra/bin/libra-node -f node.config.toml
```

See:  https://github.com/libra/libra/tree/master/config "Generating a new TestNet"

The config-builder will make the config we need. One wrinkle is that we don't know the IP address of the pods at this time, libra will NOT take a name instead of a dotted quad, so I'll use 100.100.100.100 and 101.101.101.101 etc as addresses. We create the secret key and then all the configs for every node in a folder and then copy them all into a k8s ConfigMap which we can mount as a directory in the pod. Then write some bash to replace the addresses with the real addresses and then call ```/opt/libra/bin/libra-node -f node.config.toml ```

```kubeclt get po ```  will show:

```
NAME      READY   STATUS    RESTARTS   AGE
libra-0   1/1     Running   0          12m
libra-1   1/1     Running   0          11m
libra-2   1/1     Running   0          11m
```
which looks good.  

### To test

Port forward a pod: ```kubectl port-forward libra-0 8000:8000 ```

and then, in a different terminal window, cd to the libra project and

 ```cargo run --bin cli -- -a localhost -p 8000 -m "/Users/awootton/Documents/workspace/libra-statefulset/tmp/mint.key"```

and change the path to the mint. When you get the ```libra%``` prompt type

 ```query tr 0 1 false```

which will confirm the zeroth transaction (see https://github.com/mikeholenderski/libra-docker)

Also, try this: https://developers.libra.org/docs/my-first-transaction which also works. 
