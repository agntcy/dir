# Directory SDK Models

Directory models are distributed via `buf.build` and generated from Protocol Buffers definitions,
which can become cumbersome to import and use.
This module simplifies the imports and usage of data models needed by Directory APIs.
It re-exports all the models from the generated code into dedicated namespaces so that they can be imported directly from this module.

For example, instead of importing `RecordMeta` from the generated code, use:

```python
from agntcy.dir_sdk.models.core_v1 import RecordMeta
```
