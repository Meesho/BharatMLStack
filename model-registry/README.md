### Directory Structure on Object Store
```
.
└── /bharatmlstack/
    └── /projects/
        └── /<projectA>/
            └── /modelA1/
                ├── /checkpoints/
                │   ├── /ma1.epoch1.ckpt
                │   ├── /ma1.epoch2.ckpt
                │   └── /ma3.epoch3.ckpt
                ├── /base-model/
                │   ├── /ma1.pt (.pth)
                │   └── /tensorRT/
                │       └── /ma1.plan
                ├── /quantized-1bit/
                │   ├── /ma1.pt(.pth)
                │   └── /tensorRT/
                │       └── /ma1.plan
                ├── /quantized-2bit/
                │   ├── /ma1.pt(.pth)
                │   └── /tensorRT/
                │       └── /ma1.plan
                ├── /quantized-4bit/
                │   ├── /ma1.pt(.pth)
                │   └── /tensorRT/
                │       └── /ma1.plan
                ├── /quantized-8bit/
                │   ├── /ma1.pt(.pth)
                │   └── /tensorRT/
                │       └── /ma1.plan
                └── /quantized-16bit/
                    ├── /ma1.pt(.pth)
                    └── /tensorRT/
                        └── /ma1.plan
```