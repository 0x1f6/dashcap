## Feature: Percentage-based disk safety check

The ring manager must refuse to start when pre-allocating segments would
leave less than the configured safety margin on disk. The effective margin
is the **larger** of the absolute byte threshold and the percentage-based
threshold.

### Scenario: Absolute threshold is larger than percentage threshold (small partition)

```
Given a partition with 15 GB total space
  And 5 GB free bytes
  And MinFreeAfterAlloc = 1 GB
  And MinFreePercent = 5          # → 750 MB
  And the ring requires 3 GB
When the ring manager is created
Then the effective safety margin is 1 GB        # max(1 GB, 750 MB)
  And creation succeeds                         # 5 GB ≥ 3 GB + 1 GB
```

### Scenario: Percentage threshold is larger than absolute threshold (large partition)

```
Given a partition with 100 GB total space
  And 8 GB free bytes
  And MinFreeAfterAlloc = 1 GB
  And MinFreePercent = 5          # → 5 GB
  And the ring requires 2 GB
When the ring manager is created
Then the effective safety margin is 5 GB        # max(1 GB, 5 GB)
  And creation succeeds                         # 8 GB ≥ 2 GB + 5 GB
```

### Scenario: Percentage threshold triggers rejection on large partition

```
Given a partition with 100 GB total space
  And 6 GB free bytes
  And MinFreeAfterAlloc = 1 GB
  And MinFreePercent = 5          # → 5 GB
  And the ring requires 2 GB
When the ring manager is created
Then the effective safety margin is 5 GB        # max(1 GB, 5 GB)
  And creation fails with an insufficient disk space error
  # 6 GB < 2 GB + 5 GB
```

### Scenario: Absolute threshold triggers rejection (unchanged behavior)

```
Given a partition with 15 GB total space
  And 1 GB free bytes
  And MinFreeAfterAlloc = 1 GB
  And MinFreePercent = 5          # → 750 MB
  And the ring requires 512 MB
When the ring manager is created
Then the effective safety margin is 1 GB        # max(1 GB, 750 MB)
  And creation fails with an insufficient disk space error
  # 1 GB < 512 MB + 1 GB
```

### Scenario: TotalBytes is available via DiskOps

```
Given a DiskOps implementation for Unix
When TotalBytes is called with a valid path
Then it returns the total partition size in bytes

Given a DiskOps implementation for Windows
When TotalBytes is called with a valid path
Then it returns the total partition size in bytes
```
