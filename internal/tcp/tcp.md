# TCP Server


## Shutdown

When the application is terminated the server must be able to graceful shutdown the server
and closing all connections

- Each peer must have its onw context and cancel func
- Does it make sense to use a simple channel along with signal.NotifyContext()?
- Use a waitgroup to wait for peer to stop
