# gitseer

Yet another git secrets scanner. Not geared for git actions, but more of a
continuous scanner. Hugely inspired by [N0MoreSecr3ts/wraith][1]. Anyway, this
project is not production ready and most

## Install

```
go install github.com/circleous/gitseer
```

## Configuration and Signatures

Default updated configuration and signatures can be found in [examples/][2]
directory.

```toml
max_worker = 10
with_fork = false
database = "file:gitseer.sqlite"
storage_type = "memory"
signature_path = "signatures.toml"

[[organization]]
type = "github"
name = "gojek"
expand_user = true
expand_user_fuzzy = true
```

## FAQ

> Q: Why so slow?
>
> /shrug. PR welcome though.

> Q: How to extend the signatures?
>
> You can take a look at how signatures defined in examples/signatures.toml.
> Currently, only "content" and "path" type are using regex. extension checked
> with strings.HasSuffix and filename checked with filepath.Match.

## Todo
 - [x] Detect signatures in file
 - [ ] Database, (?, somewhat works, but I still don't like it, design wise) 
 - [ ] Process only "patched" files in commits (!, a bug in go-git upstream)

[1]: https://github.com/N0MoreSecr3ts/wraith
[2]: https://github.com/circleous/gitseer/tree/main/examples
