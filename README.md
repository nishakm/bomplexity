# bomplexity

Demos around Software Bills of Materials (SBOMs).

| Folder   | Contents                                                              |
|----------|-----------------------------------------------------------------------|
| `goapp/` | Go guestbook API plus tooling that generates its CycloneDX SBOMs. See `goapp/README.md`. |

## Talk Outline

1. What is an SBOM and why is it useful for security
  - A Software Bill of Materials is a list of software components that are used to create a new Software component. See the `goapp/` Go Application for a demo of what this would look like for a single app.
  - Each software component has a list of properties that will help us with several security operations. Note that not all these properties are available from the build system.
  - A key feature of an SBOM is the encoding of the dependency tree. You'll notice for the `goapp`, there is one direct dependency, which has some transitive dependencies. Not all build systems or package management systems provide this dependency tree.
  - An SBOM is considered static for every software build. That means as soon as you rebuild or re-release software, the SBOM needs to be re-generated as well in order to capture any changes that got introduced in any of the dependencies. Comparing SBOMs between releases is a good way of tracking "drift".

2. What happens when we containerize?
  - The typical containerization method is to use Docker. We write a multistage Dockerfile for this. However, note that once an image gets built, the number of components immediately bloats up.
