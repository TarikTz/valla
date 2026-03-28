FROM eclipse-temurin:21-jdk AS builder
WORKDIR /app
COPY build.gradle settings.gradle gradlew ./
COPY gradle ./gradle
COPY src ./src
RUN ./gradlew build -DskipTests --no-daemon 2>/dev/null || gradle build -DskipTests --no-daemon

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=builder /app/build/quarkus-app/lib/ ./lib/
COPY --from=builder /app/build/quarkus-app/*.jar ./
COPY --from=builder /app/build/quarkus-app/app/ ./app/
COPY --from=builder /app/build/quarkus-app/quarkus/ ./quarkus/
EXPOSE 8080
CMD ["java", "-jar", "quarkus-run.jar"]
