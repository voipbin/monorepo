import React, { Suspense } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { UserInfoProvider } from './context/UserContext';
import './scss/style.scss';

const loading = (
    <div className="pt-3 text-center">
        <div className="sk-spinner sk-spinner-pulse"></div>
    </div>
);

// Containers
const DefaultLayout = React.lazy(() => import('./layout/DefaultLayout'));

// Pages
const Login = React.lazy(() => import('./views/pages/login/Login'));
const Register = React.lazy(() => import('./views/pages/register/Register'))

function App() {
    return (
        // <HashRouter>
        <UserInfoProvider>
            <BrowserRouter>
                <Suspense fallback={loading}>
                <Routes>
                    {/* <Route exact path="/login" name="Login Page" element={<Login />} />
            <Route exact path="/register" name="Register Page" element={<Register />} />
            <Route exact path="/404" name="Page 404" element={<Page404 />} />
            <Route exact path="/500" name="Page 500" element={<Page500 />} /> */}
                    {/* <Route path="/home/*" name="Home" element={<DefaultLayout />} /> */}
                    <Route exact path="/login" name="Login" element={<Login />} />
                    <Route path="/*" name="Home" element={<DefaultLayout />} />
                    <Route path="/dashboard" name="Home" element={<DefaultLayout />} />
                </Routes>
                </Suspense>
            </BrowserRouter>
        </UserInfoProvider>
    );
}

export default App;
