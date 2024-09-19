# kubeq

This small Go program acts as a custodian for your kubectl queries (gets only). Motivation: jsonpath is hard (and has bugs) jq syntax is even harder. When you spent couple of hours crafting your query you want
to put it somewhere. This is where kubeq comes in. It lets you save common queries in home folder and use
them by name.

## Requirements

* kubectl
* jq


## How it works

During startup kubeq checks for presence of presets file in home directory. For linux $HOME and for 
windows %APPDATA% is checked for .kubequery file. For the first time it creates the file
with defaults which you can edit later.

In a nutshell the tool does the same as:

```
kubectl get all -o json | jq <<expandedQuery>>
```

If you want to override all subcommand then use -x resourceType. Example

```
kubeq -x roles
```

becomes

``` 
kubectl get roles ...
```

## Presets

This is the defaults embedded in binary:

```json


{
    "shorts": {
        "podImage": ".spec.containers[]?.image",
        "resourceName": ".metadata.name",
        "podMountPath": ".spec.containers[]?.volumeMounts[]?.mountPath",
        "serviceTargetPort": ".spec.ports[]?.targetPort",
        "servicePort": ".spec.ports[]?.port"
    },
    
    "selects": {
        "all": ".",
        "names": "$resourceName",
        "info": "{ \"name\": $resourceName, \"metadata\": .metadata}"
    },

    "filters": {
        "id":                    "select(.)",
        "podsForImage":          "select(.spec.containers) | select($podImage | test(\"$1\"))",
		"podsForName":           "select($resourceName | test(\"$1\"))",
		"podsForLabel":          "select(.metadata.labels[\"$1\"] == \"$2\")",
		"podsForMountPath":      "select($podMountPath == \"$1\")",
		"podsForReadinessProbe": "select(.spec.containers[]?.readinessProbe.httpGet.path == \"$1\" and .spec.containers[]?.readinessProbe.httpGet.port == $2)",
		"servicesForTargetPort": "select($serviceTargetPort == $1)",
		"servicesForPort":       "select($servicePort == $1)",
		"servicesForSelector":   "select(.spec.selector[\"$1\"] == \"$2\")"
    }
}
```

**shorts**: Kubectl resources have lengthy paths, so this is where you can write shortcuts. The keys are expanded in selects and filters.

**selects**: when you specify ```-s``` flag you can select what is picked out for each object in list.

**filters** this part is how you exclude uninteresting parts of the .items[]. They are specified as functions and parameters $1, $2 in definition are args passed.

Examples:

```
kubeq -s names -w 'podsForImage(image)'
```

```
kubeq -s name -n <namespace> -w 'podsForName(name)'
```

```
kubeq -w 'podsForLabel(labelLeft,labelRight)'
```

```
kubeq -w 'podsForMountPath(path)'
```

```
kubeq -s info -w 'podsForReadinessProbe(/path, port)'
```

```
kubeq -x services  -w 'servicesForTargetPort(port)'
```

```
kubeq -x services  -w 'servicesForPort(port)'
```

```
kubeq -x services -w 'servicesForSelector(labelLeft, labelRight)'
```

```
kubeq -x roles  -w 'rolesByResourceName(name)'
```

## Hacking

True power comes from editing .kubequery file and customizing the defined queries.

## Binaries

Refer to artifacts of this [build](https://github.com/dgawlik/kubeq/actions/runs/10946695711)