# Kubernetes xDS server
This small server can be used for:
- gRPC client-side Load Balancing
- Envoy cluster CDS

Configuration in the client (supported clients as of september 2020: C-core, Java & Go) goes like this:

- 1. Set the environment variable `GRPC_XDS_BOOTSTRAP=/bootstrap.json` ([doc](https://github.com/grpc/grpc-go/tree/master/examples/features/xds))
- 2. Add this file `/bootstrap.json` containing these contents:
     
    ```json
    {
      "xds_servers": [{
        "server_uri": "xds-server-address:8080"
      }],
      "node": {
        "id": "$HOSTNAME",
        "metadata": {
            "SOME_KEY": "SOME_VALUE"
        },
        "locality": {
            "zone": "europe-west4-a"
        }
      }
    } 
    ```
- 3. Create the gRPC client like this:
    ```go
    import (
      _ "google.golang.org/grpc/xds" // To install the xds resolvers and balancers.
    )
  
    grpc.Dial("xds:///my-upstream-server", grpc.WithInsecure())
    ```

## References
1. Original proposal: https://github.com/grpc/proposal/blob/master/A27-xds-global-load-balancing.md
2. https://itnext.io/proxyless-grpc-load-balancing-in-kubernetes-ca1a4797b742 & https://github.com/asishrs/proxyless-grpc-lb
3. [GoogleBlog: efficient multi-zone and topology aware routing](https://opensource.googleblog.com/2020/11/kubernetes-efficient-multi-zone.html)