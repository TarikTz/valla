FROM mcr.microsoft.com/dotnet/sdk:8.0 AS builder
WORKDIR /app
COPY . .
RUN dotnet publish -c Release -o /app/publish

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=builder /app/publish .
EXPOSE 5000
CMD dotnet $(ls *.runtimeconfig.json | head -1 | sed 's/\.runtimeconfig\.json/\.dll/')
