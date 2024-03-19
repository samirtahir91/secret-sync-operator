kubebuilder init --domain samir.io --repo secret-sync-operator
kubebuilder create api --group sync --version v1 --kind SecretSync
make manifests
make install
make run


Set the environment variables with eval $(minikube docker-env)
Build the image with the Docker daemon of Minikube (e.g., docker build -t my-image .)
Set the image in the pod specification like the build tag (e.g., my-image)
Set the imagePullPolicy to Never, otherwise Kubernetes will try to download the image.
Important note: You have to run eval $(minikube docker-env) on each terminal you want to use, since it only sets the environment variables for the current shell session.