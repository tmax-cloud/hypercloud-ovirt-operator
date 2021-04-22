## Documentation

Edit the 'ovirt_secret.yaml' file to start ovirt operator.

```
apiVersion: v1
kind: Secret
metadata:
  name: ovirt-master
type: <Secret type>
stringData:
  url:  <URL of Ovirt Master>
  name: <Ovirt Master user name>
  pass: <Ovirt Master user password>
```

## Reporting Bugs

Use the issue tracker in this repository to report bugs.
