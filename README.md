# eventDriven

eventDriven is a Go library inspired by Watermill.
In comparison to Watermill, eventDriven focuses on providing the most essential and versatile functionalities. 
The code is more streamlined to ensure better scalability and usability.
By simplifying the interface and enhancing flexibility, it becomes more easily integrable into various projects.
More complex usage scenarios should be designed by developers. 
This way, developers can more effortlessly leverage eventDriven to build small to medium-sized event-driven applications.

## Goals

* **Concise**, **user-friendly**, and **reliable**.
* It only provides **basic** and **general** functionalities, leaving more specialized scenarios to be implemented by developers.
* As a _package_, it seamlessly integrates into your project rather than imposing itself as a rigid framework.

## What is a watermill?

> Watermill is a Go library for working efficiently with message streams. It is intended for building event
> driven applications, enabling event sourcing, RPC over messages, sagas and basically whatever else comes to
> your mind. You can use conventional pub/sub implementations like Kafka or RabbitMQ, but also HTTP or
> MySQL binlog if that fits your use case.

### Why not use Watermill directly and instead choose to develop a project with similar functionality?

Watermill is an excellent event-driven library. When I first encountered Watermill, it felt like discovering a treasure 🤩.
However, during the actual usage, I gradually discovered various mismatches between Watermill and my project.
Many features in Watermill were not suitable for my project, and due to its complexity, I couldn't encapsulate it (due to my limited experience).
If I continued using Watermill, my project would be tightly coupled with it and difficult to separate. Therefore,
to ensure outstanding scalability for my personal project, I decided to abandon the use of Watermill. Instead,
I extracted its core and most universal functionalities, made modifications, and thus created the current eventDriven.

## License

[MIT License](./LICENSE)