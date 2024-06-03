import { env } from "./config";
import { addSocket } from "./routing";
import { app } from "./server";

const server = app.listen(env.PORT, () => {
  console.log(`Server is running on port ${env.PORT}`);
});

addSocket(server);

const onCloseSignal = () => {
  server.close(() => {
    console.log("Server closed");
    process.exit(0);
  });
  setTimeout(() => process.exit(1), 10000).unref(); // Force shutdown after 10s
};

process.on("SIGINT", onCloseSignal);
process.on("SIGTERM", onCloseSignal);
