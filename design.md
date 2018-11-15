This is a simple web service that given the following request:

```sh
GET /urlinfo/1/{hostname_and_port}/{original_path_and_query_string}
```

it'll return the category of the website and whether or not it's safe to access
it.

The design is very simple. It organizes the known URLs and their information in
two level of mapping data structures that forms the url cache.  The first level
is a hash table that put URLs in the buckets with `hostname_and_port` as hash
key. In the second level, it uses a map with `hostname_and_port` and
`original_path_and_query_string` as key to locate the URL's information.

Each bucket in the hash table maintains a hit count that tallies how many times
the bucket has been hit by user requests. In the event that the number of URLs
in the cache has reached the system allowed maximum, it finds a bucket with the
least hit count, and vacates the URLs in that bucket to a file on the disk.
When a bucket with its URLs on the disk is hit by a user request, it might vacate
another bucket with the least hit count, and load the bucket into the cache. 
It's just a very basic implementation for now. For example, it doesn't take care
of the case where a bucket is loaded from the disk, and then immediately vacated
again due to not enough of hits.

When the app gets started, it loads URLs from a directory into the URL cache.
Each URL configuration file in that directory is a json file. There can be as
many configuration files as the underlying system allows. The app watches any
change in the directory and loads new/changed configuration files.

A few things to note:

1. It assumes that `original_path_and_query_string` is just a string and doesn't
   process its content.
1. Given the amount of data (URLs) quickly available for a little POC that just
   works, the code assumues 31 buckets and 100 URLs cache capacity.
1. It uses configuration files. It could have used a web interface for
   configuration.
1. When running multiple instances, one of the better choices for configuration
   files is to store them in shared persistent volumes under K8s.
1. In a matter of fact, it could have used some sort of key/value config stores
   such as memcached, etcd, .... (there is a long list to choose depending on
   the requirement) for storing the URLs information. Updates, memory
   limitation, etc, can be taken care of without reinventing the wheels.
