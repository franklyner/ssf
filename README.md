# ssf
Small Server Framework: Lightweight framework for building web and api servers with GO

## Goals
The main goal is as usual to avoid boiler plate code, enforce some standards and provide some functionalities out-of-the-box that just take care of a few things. Also, we want to promote a clean code structure.

## High-Level Approach
The most central concept is that the part of the code that instantiates the server (typically the place where your func main() lives) takes care of wiring up all dependencies. It aligns itself to the concept of an application controller.

The we introduce 3 main parts:
### Controller
A controller is the code that actually handles requests for a specific route. It declares all of the required properties and provide an auth funtion pointer if secured and a controller function pointer.. the code that actually is executed for the request.

Typically you need a bunch of related controllers (such as one for each CRUD operatio for a rest API). To simplify things for the application controller, all related controllers are registered to the server through a ControllerProvider. This allows to nicely organize the different aspects of an application into differen files and provide one controller provider for each file and the controllers therein. This promotes in our opinion a clean code structure.

### Context
For each request a context is instantiated. The context provides access to the request and provides all required functions to send back the response. It also provides access to the service registry.

### Service
A service is nothing else than a struct instance the provides some functions. It is registered to the server by the application controller. On each request the service registry is added to the context and passed down to the controllers for execution. This allows all controllers and inter-dependent services to access other services as long as they get access to the context.

## Configuration Concept
The framework provides a Config type. It's basically a simple map but can optionally use it's pre-implemented viper instanciation. The main guideline is that we want the server to fail upon startup in case of missing configuration. To ensure that each required config property needs to be read by the application controller. The Config functions will immediately panic in case one is missign.

This is the reason why the config object is not passed along with the context. Instead it is expected that each ControllerProvider and Service struct exports fiels for each config property required. It is the application controllers job to populate those properties either through the help of the Config object or by other means. 

Controller functions get access to those by exposing a Config property. The controller provider is responsible to instantiate the config map for each controller (if required) and add it to the controller as part of the getControllers() function.

## Free stuff
Some things come for free:
* Automatic logging of all requests, the corresponding response and measurment of the execution duration.
* Each request gets it's unique UUID. All messages logged through the functions provided by the context will be prefixed with the request id for correlation.
* Some easy to use methods to send html and json responses
* Easy testability: Ther server exposes a GetMainHandler() function that gives access to the main request handler which can then be used for unit testing.
* A status page that gives an overview of how many times each controller has been called and since when the server is running.
