import { createEnv } from "@t3-oss/env-core";
import dotenv from "dotenv";
import { z } from "zod";

dotenv.config();

export const env = createEnv({
  server: {
    NODE_ENV: z
      .literal("development")
      .or(z.literal("production"))
      .or(z.literal("test"))
      .default("development"),
    PORT: z.number().default(4000),
    SESSION_SECRET: z.string().default("secret"),
  },
  runtimeEnv: process.env,
});
