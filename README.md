# kubeq

This tool makes filtering of resources via kubectl a little easier. It wraps kubectl and lets you
define "aliases" to most common used jsonpath queries. For filtering it uses go implementation of jq.

## How it works

During startup kubeq checks for presence of presets file in home directory. For linux $HOME and for 
windows %APPDATA% is checked for presence of .kubequery. For the first time kubeq creates the file
with defaults which you can edit later.

Then it parses the commandline args. In particular '-q <query>' is pulled out and parsed. Then it translates it to jq query with parameter substitutions. So it is equivalent of 

```
kubectl get all -o json | jq <<translatedQuery>>
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

```
kubeq -q 'podsForImage(image)'
```

```
kubeq -q 'podsForName(name)'
```

```
kubeq -q 'podsForLabel(labelLeft,labelRight)'
```

```
kubeq -q 'podsForMountPath(path)'
```

```
kubeq -q 'podsForReadinessProbe(/path, port)'
```

```
kubeq -x services  -q 'servicesForTargetPort(port)'
```

```
kubeq -x services  -q 'servicesForPort(port)'
```

```
kubeq -x services -q 'servicesForSelector(labelLeft, labelRight)'
```

```
kubeq -x roles  -q 'rolesByResourceName(name)'
```

## Hacking

True power comes from editing .kubequery file and adding your own selectors. So go ahead and customize

## Binaries

Refer to artifact of this [build](https://github.com/dgawlik/kubeq/actions/runs/10861685145)