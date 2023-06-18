# NGINX DISTRO
Distribution to the official nginx application created by me.

•	Added a sync Pool to reuse instances of variables and reduce memory allocations.
•	Changed receiver to pointers to mitigate extra copying of observer slice.
•	Implemented the httpcache package instead of rolling out a custom caching mechanism. 
•	Added mutex to protect the cache map in the Cache struct, as multiple goroutines may access it concurrently and more



