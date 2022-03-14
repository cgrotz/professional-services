VERSION=3
KO_DOCKER_REPO=europe-docker.pkg.dev/ipam-autopilot-showcase/ipam/ipam

build: main.go
	KO_DOCKER_REPO=${KO_DOCKER_REPO} ko publish ./

deploy:
	docker buildx build --platform linux/amd64 --push -t europe-docker.pkg.dev/ipam-autopilot-showcase/ipam/ipam:${VERSION} ../
	gcloud run services update ipam --image=europe-docker.pkg.dev/ipam-autopilot-showcase/ipam/ipam:${VERSION} --region=europe-west1
#	KO_DOCKER_REPO="europe-docker.pkg.dev/ipam-autopilot-showcase/ipam/ipam" gcloud run services update ipam --image=$(ko publish ./) --region=europe-west1
