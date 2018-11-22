deploy a hosted docker container to an existing ecs cluster

Allows deployment of ECS microservices straight from the command line!

## Requirements

You need:

- Valid AWS credentials for the place you're deploying to (in your ENV)
- An existing task definition - this won't create one for you

## Usage

The full list of options is:

```
Usage of ./go-ecs-deploy:
  -a string
        Application name (can be specified multiple times)
  -c string
        Cluster name to deploy to
  -d    Enable Debug output
  -i string
        Docker repo to pull from
  -r string
        AWS region
  -s string
        Tag, usually short git SHA to deploy
```

### Example

```
AWS_PROFILE=production go-ecs-deploy \
  -c cluster-name \
  -a authome \
  -i quay.io/username/reponame \
  -s 5304a1b \
  -r us-west-2
```

## Development

Run `dep ensure` to get latest dependencies.
