FROM debian:13-slim
RUN apt-get update && apt-get install -y libusb-1.0-0
ADD go-barcode-to-bcw /
CMD ["/go-barcode-to-bcw"]