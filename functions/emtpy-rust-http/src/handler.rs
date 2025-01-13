use crate::config::HandlerConfig;
use actix_web::{http::Method, web::Data, HttpRequest, HttpResponse};
use log::info;
use std::sync::Mutex;
use lazy_static::lazy_static;

// Define a static variable to track cold start
lazy_static! {
    static ref IS_COLD: Mutex<bool> = Mutex::new(true);
}

// Implement your function's logic here
pub async fn index(req: HttpRequest, config: Data<HandlerConfig>) -> HttpResponse {
    info!("{:#?}", req);

    // Check and update the cold start flag
    let mut is_cold = IS_COLD.lock().unwrap();
    let response = if *is_cold {
        "true"
    } else {
        "false"
    };
    *is_cold = false;

    HttpResponse::Ok().body(response)
}

#[cfg(test)]
mod tests {
    use super::*;
    use actix_web::{body::to_bytes, http, test::TestRequest, web::Bytes};

    fn config() -> Data<HandlerConfig> {
        Data::new(HandlerConfig::default())
    }

    #[actix_rt::test]
    async fn get() {
        let req = TestRequest::get().to_http_request();
        let resp = index(req, config()).await;
        assert_eq!(resp.status(), http::StatusCode::OK);
        assert_eq!(
            &Bytes::from("true"),
            to_bytes(resp.into_body()).await.unwrap().as_ref()
        );
    }

    #[actix_rt::test]
    async fn post() {
        let req = TestRequest::post().to_http_request();
        let resp = index(req, config()).await;
        assert!(resp.status().is_success());
        assert_eq!(
            &Bytes::from("false"),
            to_bytes(resp.into_body()).await.unwrap().as_ref()
        );
    }
}