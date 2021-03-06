Refer to the [design doc](design.md)

To build the application, the docker image and push it to the docker hub
- setup your docker HUB and TAG environment variables
- make build; make docker; make push

To deploy it in a K8s cluster
- kubectl apply -f url-lookup.yaml

There will be two Pods deployed. To find out if the apps are running:

   ```kubectl get pods -o wide | grep url-lookup```

To invoke the service
- find the url-lookup service ip: 

   ```kubectl get service```

- inside the cluster

   ```curl <url-lookup service ip>:16888/urlinfo/1/skgroup.kiev.ua:80/index.html```

- issue the above command a few times, the requests will be load-balanced to the two
  url-lookup service instances.

To see the service log

  ```kubectl logs <url-lookup-pod-id>```


Invoke url-lookup -h to find help information:

```sh
URL lookup service returns URL information including if it's safe to open.

Usage:
  url-lookup [flags]

Flags:
  -h, --help                     help for url-lookup
      --port int                 URL lookup service port (default 16888)
      --url-cache-path string    URL cache path
      --url-config-path string   URL configuration path
```
